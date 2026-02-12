package autoscaler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// SQSClient defines the interface for SQS operations needed by the PolicyEvaluator
type SQSClient interface {
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
}

// ScalingPolicy defines a queue-depth based scaling policy
type ScalingPolicy struct {
	PolicyType                string `dynamodbav:"policy_type"`                  // "queue-depth"
	QueueURL                  string `dynamodbav:"queue_url"`                    // SQS queue URL
	TargetMessagesPerInstance int    `dynamodbav:"target_messages_per_instance"` // e.g., 10
	ScaleUpCooldownSeconds    int    `dynamodbav:"scale_up_cooldown_seconds"`    // default: 60
	ScaleDownCooldownSeconds  int    `dynamodbav:"scale_down_cooldown_seconds"`  // default: 300
}

// ScalingState tracks recent scaling activity
type ScalingState struct {
	LastScaleUp            time.Time `dynamodbav:"last_scale_up,omitempty"`
	LastScaleDown          time.Time `dynamodbav:"last_scale_down,omitempty"`
	LastQueueDepth         int       `dynamodbav:"last_queue_depth"`
	LastCalculatedCapacity int       `dynamodbav:"last_calculated_capacity"`
}

// PolicyEvaluator evaluates scaling policies and calculates desired capacity
type PolicyEvaluator struct {
	sqsClient SQSClient
}

// NewPolicyEvaluator creates a new policy evaluator
func NewPolicyEvaluator(sqsClient SQSClient) *PolicyEvaluator {
	return &PolicyEvaluator{
		sqsClient: sqsClient,
	}
}

// EvaluatePolicy calculates new desired capacity based on scaling policy
// Returns: (newDesiredCapacity, queueDepth, changed, error)
func (p *PolicyEvaluator) EvaluatePolicy(
	ctx context.Context,
	group *AutoScaleGroup,
) (int, int, bool, error) {
	if group.ScalingPolicy == nil {
		return group.DesiredCapacity, 0, false, nil
	}

	// Query SQS for queue depth
	queueDepth, err := p.getQueueDepth(ctx, group.ScalingPolicy.QueueURL)
	if err != nil {
		return 0, 0, false, fmt.Errorf("get queue depth: %w", err)
	}

	// Calculate needed capacity
	needed := p.calculateDesiredCapacity(
		queueDepth,
		group.ScalingPolicy.TargetMessagesPerInstance,
	)

	// Enforce min/max bounds
	if needed < group.MinCapacity {
		needed = group.MinCapacity
	}
	if needed > group.MaxCapacity {
		needed = group.MaxCapacity
	}

	// Check if change is needed
	if needed == group.DesiredCapacity {
		return needed, queueDepth, false, nil
	}

	// Check cooldown
	scaleUp := needed > group.DesiredCapacity
	if p.inCooldown(group.ScalingState, group.ScalingPolicy, scaleUp) {
		return group.DesiredCapacity, queueDepth, false, nil
	}

	return needed, queueDepth, true, nil
}

// getQueueDepth queries SQS for total queue depth (visible + in-flight messages)
func (p *PolicyEvaluator) getQueueDepth(ctx context.Context, queueURL string) (int, error) {
	result, err := p.sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(queueURL),
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
			types.QueueAttributeNameApproximateNumberOfMessagesNotVisible,
		},
	})
	if err != nil {
		return 0, err
	}

	visible := parseIntAttr(result.Attributes, string(types.QueueAttributeNameApproximateNumberOfMessages))
	inFlight := parseIntAttr(result.Attributes, string(types.QueueAttributeNameApproximateNumberOfMessagesNotVisible))

	return visible + inFlight, nil
}

// calculateDesiredCapacity computes needed instances using ceiling division
func (p *PolicyEvaluator) calculateDesiredCapacity(queueDepth, targetMessagesPerInstance int) int {
	if queueDepth == 0 || targetMessagesPerInstance <= 0 {
		return 0
	}

	// Ceiling division: (depth + target - 1) / target
	return (queueDepth + targetMessagesPerInstance - 1) / targetMessagesPerInstance
}

// inCooldown checks if scaling action is in cooldown period
func (p *PolicyEvaluator) inCooldown(
	state *ScalingState,
	policy *ScalingPolicy,
	scaleUp bool,
) bool {
	if state == nil {
		return false
	}

	var lastScale time.Time
	var cooldownSeconds int

	if scaleUp {
		lastScale = state.LastScaleUp
		cooldownSeconds = policy.ScaleUpCooldownSeconds
	} else {
		lastScale = state.LastScaleDown
		cooldownSeconds = policy.ScaleDownCooldownSeconds
	}

	if lastScale.IsZero() {
		return false
	}

	elapsed := time.Since(lastScale)
	cooldown := time.Duration(cooldownSeconds) * time.Second

	return elapsed < cooldown
}

// parseIntAttr safely parses integer attribute from SQS response
func parseIntAttr(attrs map[string]string, key string) int {
	val, ok := attrs[key]
	if !ok {
		return 0
	}

	n, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}

	return n
}
