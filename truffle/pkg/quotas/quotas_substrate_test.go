package quotas

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/scttfrdmn/spore-host/truffle/pkg/testutil"
)

func TestGetQuotas_StandardFamily(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	c := NewClientFromConfig(env.AWSConfig)

	info, err := c.GetQuotas(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("GetQuotas() error = %v", err)
	}
	if info == nil {
		t.Fatal("GetQuotas() returned nil")
	}
	if info.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", info.Region, "us-east-1")
	}

	// Substrate seeds L-1216C47A (Standard On-Demand) = 32 vCPUs.
	v, ok := info.OnDemand[FamilyStandard]
	if !ok {
		t.Fatal("OnDemand[FamilyStandard] not populated")
	}
	if v != 32 {
		t.Errorf("OnDemand[FamilyStandard] = %d, want 32", v)
	}
}

func TestListServiceQuotas_EC2NonEmpty(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	sqc := env.ServiceQuotasClient()

	out, err := sqc.ListServiceQuotas(ctx, &servicequotas.ListServiceQuotasInput{
		ServiceCode: aws.String("ec2"),
	})
	if err != nil {
		t.Fatalf("ListServiceQuotas() error = %v", err)
	}
	if len(out.Quotas) == 0 {
		t.Fatal("ListServiceQuotas() returned 0 quotas for ec2, want >= 1")
	}
}

func TestRequestServiceQuotaIncrease_Pending(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	sqc := env.ServiceQuotasClient()

	out, err := sqc.RequestServiceQuotaIncrease(ctx, &servicequotas.RequestServiceQuotaIncreaseInput{
		ServiceCode:  aws.String("ec2"),
		QuotaCode:    aws.String(QuotaCodeStandard),
		DesiredValue: aws.Float64(256),
	})
	if err != nil {
		t.Fatalf("RequestServiceQuotaIncrease() error = %v", err)
	}
	if out.RequestedQuota == nil {
		t.Fatal("RequestedQuota is nil")
	}
	if out.RequestedQuota.Status == "" {
		t.Fatal("RequestedQuota.Status is empty")
	}
	if string(out.RequestedQuota.Status) != "PENDING" {
		t.Errorf("Status = %q, want %q", out.RequestedQuota.Status, "PENDING")
	}
}

func TestGetQuotas_Caching(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	c := NewClientFromConfig(env.AWSConfig)

	first, err := c.GetQuotas(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("GetQuotas() first call error = %v", err)
	}

	second, err := c.GetQuotas(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("GetQuotas() second call error = %v", err)
	}

	// Within cacheTTL the second call returns the same pointer.
	if first != second {
		t.Error("second GetQuotas() call did not return cached result (different pointer)")
	}
}
