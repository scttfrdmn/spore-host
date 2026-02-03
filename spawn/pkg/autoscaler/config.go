package autoscaler

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// Config holds dependencies for the autoscaler
type Config struct {
	EC2Client     *ec2.Client
	DynamoClient  *dynamodb.Client
	TableName     string
	RegistryTable string
}

// AutoScaleGroup represents an auto-scaling job array configuration
type AutoScaleGroup struct {
	AutoScaleGroupID    string
	GroupName           string
	JobArrayID          string
	DesiredCapacity     int
	MinCapacity         int
	MaxCapacity         int
	LaunchTemplate      LaunchTemplate
	Status              string // "active", "paused", "terminated"
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastScaleEvent      time.Time
	HealthCheckInterval time.Duration
	ReplacementStrategy string // "immediate", "rolling"
}

// LaunchTemplate defines how to launch new instances
type LaunchTemplate struct {
	InstanceType       string
	AMI                string
	Spot               bool
	KeyName            string
	SubnetID           string
	SecurityGroups     []string
	IAMInstanceProfile string
	UserData           string
	Tags               map[string]string
}

// HealthStatus represents the health status of an instance
type HealthStatus struct {
	InstanceID       string
	EC2State         string
	HeartbeatAge     time.Duration
	SpotInterruption bool
	Healthy          bool
	Reason           string
}

// CapacityPlan describes capacity changes to execute
type CapacityPlan struct {
	CurrentCapacity int
	DesiredCapacity int
	HealthyCount    int
	UnhealthyCount  int
	PendingCount    int
	ToLaunch        int
	ToTerminate     []string
}
