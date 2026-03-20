// Package testutil provides shared test helpers for truffle packages.
package testutil

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	substrate "github.com/scttfrdmn/substrate"
)

// TestEnv holds a running Substrate server and a pre-configured AWS config
// that points all SDK calls at the emulator.
type TestEnv struct {
	// URL is the base URL of the Substrate server.
	URL string
	// AWSConfig is a pre-configured aws.Config pointing at the Substrate server.
	AWSConfig aws.Config
}

// SubstrateServer starts a Substrate emulator and returns a TestEnv.
// The server is shut down automatically when the test ends.
func SubstrateServer(t *testing.T) *TestEnv {
	t.Helper()
	ts := substrate.StartTestServer(t)

	cfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithBaseEndpoint(ts.URL),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", "test"),
		),
	)
	if err != nil {
		t.Fatalf("SubstrateServer: build AWS config: %v", err)
	}

	return &TestEnv{URL: ts.URL, AWSConfig: cfg}
}

// EC2Client returns an EC2 client pointed at the Substrate server.
func (e *TestEnv) EC2Client() *ec2.Client {
	return ec2.NewFromConfig(e.AWSConfig)
}

// ServiceQuotasClient returns a Service Quotas client pointed at the Substrate server.
func (e *TestEnv) ServiceQuotasClient() *servicequotas.Client {
	return servicequotas.NewFromConfig(e.AWSConfig)
}
