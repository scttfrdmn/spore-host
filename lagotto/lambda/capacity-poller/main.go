package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/scttfrdmn/spore-host/lagotto/pkg/watcher"
	truffleaws "github.com/scttfrdmn/spore-host/truffle/pkg/aws"
)

var (
	cfg             aws.Config
	store           *watcher.Store
	poller          *watcher.Poller
	schedulerClient *scheduler.Client
	scheduleName    string
)

func init() {
	ctx := context.Background()

	var err error
	cfg, err = config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	watchesTable := getEnv("WATCHES_TABLE", "lagotto-watches")
	historyTable := getEnv("HISTORY_TABLE", "lagotto-match-history")
	snsTopicArn := os.Getenv("SNS_TOPIC_ARN")
	scheduleName = getEnv("SCHEDULE_NAME", "lagotto-capacity-poller")

	store = watcher.NewStore(cfg, watchesTable, historyTable)

	truffleClient, err := truffleaws.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create truffle client: %v", err)
	}

	// Set up notifier
	var notifier *watcher.Notifier
	if snsTopicArn != "" {
		notifier = watcher.NewNotifier(cfg, snsTopicArn)
	}

	// Set up spawner (for action=spawn watches)
	spawner, err := watcher.NewSpawner(ctx)
	if err != nil {
		log.Printf("Warning: auto-spawn unavailable: %v", err)
	}

	poller = watcher.NewPoller(truffleClient, store, true, watcher.PollerOpts{
		Notifier: notifier,
		Spawner:  spawner,
	})

	schedulerClient = scheduler.NewFromConfig(cfg)
}

func handler(ctx context.Context) error {
	log.Println("Starting capacity poll cycle")

	matches, err := poller.PollAll(ctx)
	if err != nil {
		return fmt.Errorf("poll: %w", err)
	}

	log.Printf("Poll complete: %d matches found", len(matches))

	// Check if any active watches remain
	active, err := store.ListActiveWatches(ctx)
	if err != nil {
		log.Printf("Warning: failed to check active watches: %v", err)
		return nil
	}

	if len(active) == 0 {
		log.Println("No active watches remaining, disabling schedule")
		if err := disableSchedule(ctx); err != nil {
			log.Printf("Warning: failed to disable schedule: %v", err)
		}
	} else {
		log.Printf("%d active watches remaining", len(active))
	}

	return nil
}

func disableSchedule(ctx context.Context) error {
	current, err := schedulerClient.GetSchedule(ctx, &scheduler.GetScheduleInput{
		Name: aws.String(scheduleName),
	})
	if err != nil {
		return fmt.Errorf("get schedule: %w", err)
	}

	_, err = schedulerClient.UpdateSchedule(ctx, &scheduler.UpdateScheduleInput{
		Name:               current.Name,
		ScheduleExpression: current.ScheduleExpression,
		FlexibleTimeWindow: current.FlexibleTimeWindow,
		Target:             current.Target,
		State:              schedulertypes.ScheduleStateDisabled,
	})
	if err != nil {
		return fmt.Errorf("disable schedule: %w", err)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	lambda.Start(handler)
}
