package autoscaler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Mock SQS client for testing
type mockSQSClient struct {
	queueDepth int
	err        error
}

func (m *mockSQSClient) GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Split queue depth between visible and in-flight (arbitrary split for testing)
	visible := m.queueDepth / 2
	inFlight := m.queueDepth - visible

	return &sqs.GetQueueAttributesOutput{
		Attributes: map[string]string{
			string(types.QueueAttributeNameApproximateNumberOfMessages):           fmt.Sprintf("%d", visible),
			string(types.QueueAttributeNameApproximateNumberOfMessagesNotVisible): fmt.Sprintf("%d", inFlight),
		},
	}, nil
}

func TestCalculateDesiredCapacity(t *testing.T) {
	tests := []struct {
		name       string
		queueDepth int
		target     int
		want       int
	}{
		{"empty queue", 0, 10, 0},
		{"partial load", 50, 10, 5},
		{"exact multiple", 100, 10, 10},
		{"ceiling division", 105, 10, 11},
		{"single message", 1, 10, 1},
		{"target is zero", 50, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := &PolicyEvaluator{}
			got := pe.calculateDesiredCapacity(tt.queueDepth, tt.target)

			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEvaluatePolicy_MinMaxBounds(t *testing.T) {
	tests := []struct {
		name        string
		queueDepth  int
		target      int
		min         int
		max         int
		current     int
		wantDesired int
		wantChanged bool
	}{
		{
			name:        "clamped to min",
			queueDepth:  5,
			target:      10,
			min:         2,
			max:         20,
			current:     2,
			wantDesired: 2,
			wantChanged: false,
		},
		{
			name:        "clamped to max",
			queueDepth:  300,
			target:      10,
			min:         0,
			max:         15,
			current:     10,
			wantDesired: 15,
			wantChanged: true,
		},
		{
			name:        "within bounds",
			queueDepth:  50,
			target:      10,
			min:         0,
			max:         20,
			current:     0,
			wantDesired: 5,
			wantChanged: true,
		},
		{
			name:        "no change needed",
			queueDepth:  50,
			target:      10,
			min:         0,
			max:         20,
			current:     5,
			wantDesired: 5,
			wantChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock SQS client
			mockSQS := &mockSQSClient{queueDepth: tt.queueDepth}
			pe := NewPolicyEvaluator(mockSQS)

			group := &AutoScaleGroup{
				DesiredCapacity: tt.current,
				MinCapacity:     tt.min,
				MaxCapacity:     tt.max,
				ScalingPolicy: &ScalingPolicy{
					PolicyType:                "queue-depth",
					QueueURL:                  "https://sqs.us-east-1.amazonaws.com/test/queue",
					TargetMessagesPerInstance: tt.target,
					ScaleUpCooldownSeconds:    60,
					ScaleDownCooldownSeconds:  300,
				},
			}

			gotDesired, _, gotChanged, err := pe.EvaluatePolicy(context.Background(), group)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if gotDesired != tt.wantDesired {
				t.Errorf("desired capacity: got %d, want %d", gotDesired, tt.wantDesired)
			}

			if gotChanged != tt.wantChanged {
				t.Errorf("changed: got %v, want %v", gotChanged, tt.wantChanged)
			}
		})
	}
}

func TestInCooldown(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		state   *ScalingState
		policy  *ScalingPolicy
		scaleUp bool
		want    bool
	}{
		{
			name:    "no state - not in cooldown",
			state:   nil,
			policy:  &ScalingPolicy{ScaleUpCooldownSeconds: 60, ScaleDownCooldownSeconds: 300},
			scaleUp: true,
			want:    false,
		},
		{
			name: "scale up - in cooldown",
			state: &ScalingState{
				LastScaleUp: now.Add(-30 * time.Second),
			},
			policy:  &ScalingPolicy{ScaleUpCooldownSeconds: 60, ScaleDownCooldownSeconds: 300},
			scaleUp: true,
			want:    true,
		},
		{
			name: "scale up - cooldown expired",
			state: &ScalingState{
				LastScaleUp: now.Add(-90 * time.Second),
			},
			policy:  &ScalingPolicy{ScaleUpCooldownSeconds: 60, ScaleDownCooldownSeconds: 300},
			scaleUp: true,
			want:    false,
		},
		{
			name: "scale down - in cooldown",
			state: &ScalingState{
				LastScaleDown: now.Add(-2 * time.Minute),
			},
			policy:  &ScalingPolicy{ScaleUpCooldownSeconds: 60, ScaleDownCooldownSeconds: 300},
			scaleUp: false,
			want:    true,
		},
		{
			name: "scale down - cooldown expired",
			state: &ScalingState{
				LastScaleDown: now.Add(-6 * time.Minute),
			},
			policy:  &ScalingPolicy{ScaleUpCooldownSeconds: 60, ScaleDownCooldownSeconds: 300},
			scaleUp: false,
			want:    false,
		},
		{
			name: "scale up during scale down cooldown - allowed",
			state: &ScalingState{
				LastScaleDown: now.Add(-1 * time.Minute),
			},
			policy:  &ScalingPolicy{ScaleUpCooldownSeconds: 60, ScaleDownCooldownSeconds: 300},
			scaleUp: true,
			want:    false,
		},
		{
			name: "no previous scale event",
			state: &ScalingState{
				LastQueueDepth: 50,
			},
			policy:  &ScalingPolicy{ScaleUpCooldownSeconds: 60, ScaleDownCooldownSeconds: 300},
			scaleUp: true,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := &PolicyEvaluator{}
			got := pe.inCooldown(tt.state, tt.policy, tt.scaleUp)

			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluatePolicy_NilPolicy(t *testing.T) {
	pe := &PolicyEvaluator{}

	group := &AutoScaleGroup{
		DesiredCapacity: 5,
		ScalingPolicy:   nil,
	}

	gotDesired, gotQueueDepth, gotChanged, err := pe.EvaluatePolicy(context.Background(), group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotDesired != 5 {
		t.Errorf("desired capacity: got %d, want 5", gotDesired)
	}

	if gotQueueDepth != 0 {
		t.Errorf("queue depth: got %d, want 0", gotQueueDepth)
	}

	if gotChanged {
		t.Errorf("changed: got true, want false")
	}
}

func TestParseIntAttr(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
		key   string
		want  int
	}{
		{"valid integer", map[string]string{"foo": "42"}, "foo", 42},
		{"missing key", map[string]string{"foo": "42"}, "bar", 0},
		{"invalid integer", map[string]string{"foo": "abc"}, "foo", 0},
		{"empty string", map[string]string{"foo": ""}, "foo", 0},
		{"zero", map[string]string{"foo": "0"}, "foo", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIntAttr(tt.attrs, tt.key)

			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
