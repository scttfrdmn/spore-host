package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Holder creates On-Demand Capacity Reservations to hold capacity.
type Holder struct {
	cfg aws.Config
}

// NewHolder creates a Holder.
func NewHolder(cfg aws.Config) *Holder {
	return &Holder{cfg: cfg}
}

// Hold creates a targeted ODCR for the matched instance type and AZ.
// The reservation expires after 30 minutes — the user must launch within that window.
// Updates the MatchResult with the reservation ID on success.
func (h *Holder) Hold(ctx context.Context, w *Watch, m *MatchResult) error {
	if m.AvailabilityZone == "" {
		return fmt.Errorf("cannot create capacity reservation without availability zone")
	}

	// Create a region-specific EC2 client
	cfg := h.cfg.Copy()
	cfg.Region = m.Region
	client := ec2.NewFromConfig(cfg)

	endDate := time.Now().UTC().Add(30 * time.Minute)

	result, err := client.CreateCapacityReservation(ctx, &ec2.CreateCapacityReservationInput{
		InstanceType:          aws.String(m.InstanceType),
		InstancePlatform:      ec2types.CapacityReservationInstancePlatformLinuxUnix,
		AvailabilityZone:      aws.String(m.AvailabilityZone),
		InstanceCount:         aws.Int32(1),
		EndDateType:           ec2types.EndDateTypeLimited,
		EndDate:               aws.Time(endDate),
		InstanceMatchCriteria: ec2types.InstanceMatchCriteriaTargeted,
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeCapacityReservation,
				Tags: []ec2types.Tag{
					{Key: aws.String("lagotto:watch-id"), Value: aws.String(w.WatchID)},
					{Key: aws.String("lagotto:managed"), Value: aws.String("true")},
				},
			},
		},
	})
	if err != nil {
		m.ActionTaken = "hold_failed"
		return fmt.Errorf("create capacity reservation: %w", err)
	}

	m.ReservationID = *result.CapacityReservation.CapacityReservationId
	m.ActionTaken = "held"
	return nil
}
