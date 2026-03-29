package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
)

var (
	queueOutputDir string
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage batch job queues",
	Long: `Commands for managing and monitoring batch job queues.

Batch queues execute jobs sequentially on a single instance, with
dependency management and automatic result collection.

Examples:
  # Check queue status on instance
  spawn queue status i-1234567890abcdef0

  # Download queue results
  spawn queue results queue-20260122-140530 --output ./results/
`,
}

var queueStatusCmd = &cobra.Command{
	Use:   "status <instance-id>",
	Short: "Show queue execution status",
	Long: `Show the execution status of a batch queue running on an instance.

Connects to the instance via SSH and reads the queue state file.

Examples:
  spawn queue status i-1234567890abcdef0
`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueStatus,
}

var queueResultsCmd = &cobra.Command{
	Use:   "results <queue-id>",
	Short: "Download queue results from S3",
	Long: `Download all job results from S3 for a completed or running queue.

Results include job outputs, logs, and the final queue state.

Examples:
  # Download to current directory
  spawn queue results queue-20260122-140530

  # Download to specific directory
  spawn queue results queue-20260122-140530 --output ./my-results/
`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueResults,
}

func init() {
	// Results subcommand flags
	queueResultsCmd.Flags().StringVarP(&queueOutputDir, "output", "o", ".", "Output directory for results")

	// Add subcommands
	queueCmd.AddCommand(queueStatusCmd)
	queueCmd.AddCommand(queueResultsCmd)

	rootCmd.AddCommand(queueCmd)
}

func runQueueStatus(cmd *cobra.Command, args []string) error {
	instanceID := args[0]

	fmt.Fprintf(os.Stderr, "\n📊 Queue Status\n")
	fmt.Fprintf(os.Stderr, "   Instance: %s\n\n", instanceID)

	// SSH to instance and read state file
	fmt.Fprintf(os.Stderr, "Connecting to instance...\n")

	sshCmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("ec2-user@%s", instanceID),
		"cat /var/lib/spored/queue-state.json")

	output, err := sshCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to read queue state: %w\nOutput: %s", err, string(output))
	}

	// Parse state
	var state struct {
		QueueID   string    `json:"queue_id"`
		StartedAt string    `json:"started_at"`
		UpdatedAt string    `json:"updated_at"`
		Status    string    `json:"status"`
		Jobs      []struct {
			JobID           string `json:"job_id"`
			Status          string `json:"status"`
			StartedAt       string `json:"started_at,omitempty"`
			CompletedAt     string `json:"completed_at,omitempty"`
			ExitCode        int    `json:"exit_code,omitempty"`
			Attempt         int    `json:"attempt"`
			PID             int    `json:"pid,omitempty"`
			ErrorMessage    string `json:"error_message,omitempty"`
			ResultsUploaded bool   `json:"results_uploaded"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(output, &state); err != nil {
		return fmt.Errorf("failed to parse queue state: %w", err)
	}

	// Display status
	fmt.Fprintf(os.Stderr, "\nQueue ID:    %s\n", state.QueueID)
	fmt.Fprintf(os.Stderr, "Status:      %s\n", state.Status)
	fmt.Fprintf(os.Stderr, "Started:     %s\n", state.StartedAt)
	fmt.Fprintf(os.Stderr, "Updated:     %s\n", state.UpdatedAt)
	fmt.Fprintf(os.Stderr, "\n")

	// Display jobs
	fmt.Fprintf(os.Stderr, "Jobs:\n")
	fmt.Fprintf(os.Stderr, "%-25s %-12s %-8s %-8s %s\n", "JOB ID", "STATUS", "ATTEMPT", "EXIT", "RESULTS")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 80))

	for _, job := range state.Jobs {
		exitCode := "-"
		if job.Status == "completed" || job.Status == "failed" {
			exitCode = fmt.Sprintf("%d", job.ExitCode)
		}

		results := "no"
		if job.ResultsUploaded {
			results = "yes"
		}

		pid := ""
		if job.PID > 0 {
			pid = fmt.Sprintf("(PID: %d)", job.PID)
		}

		fmt.Fprintf(os.Stderr, "%-25s %-12s %-8d %-8s %-8s %s\n",
			job.JobID, job.Status, job.Attempt, exitCode, results, pid)

		if job.ErrorMessage != "" {
			fmt.Fprintf(os.Stderr, "  Error: %s\n", job.ErrorMessage)
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
	return nil
}

func runQueueResults(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	queueID := args[0]

	fmt.Fprintf(os.Stderr, "\n📦 Downloading Queue Results\n")
	fmt.Fprintf(os.Stderr, "   Queue ID: %s\n", queueID)
	fmt.Fprintf(os.Stderr, "   Output:   %s\n\n", queueOutputDir)

	// Determine region (use first from regions slice or default)
	region := "us-east-1"
	if len(regions) > 0 {
		region = regions[0]
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithSharedConfigProfile("spore-host-dev"),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Determine S3 bucket and prefix
	// This should match what was specified in the queue config
	bucket := fmt.Sprintf("spawn-results-%s", region)
	prefix := fmt.Sprintf("queues/%s/", queueID)

	// Create output directory
	if err := os.MkdirAll(queueOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// List all objects
	fmt.Fprintf(os.Stderr, "Listing results...\n")
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	fileCount := 0
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			fileCount++

			// Download object
			localPath := filepath.Join(queueOutputDir, strings.TrimPrefix(*obj.Key, prefix))
			localDir := filepath.Dir(localPath)

			if err := os.MkdirAll(localDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			fmt.Fprintf(os.Stderr, "  Downloading: %s\n", strings.TrimPrefix(*obj.Key, prefix))

			result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    obj.Key,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: failed to download: %v\n", err)
				continue
			}

			// Write to file
			outFile, err := os.Create(localPath)
			if err != nil {
				_ = result.Body.Close()
				return fmt.Errorf("failed to create file: %w", err)
			}

			_, err = outFile.ReadFrom(result.Body)
			_ = result.Body.Close()
			_ = outFile.Close()

			if err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n✅ Downloaded %d files to %s\n\n", fileCount, queueOutputDir)
	return nil
}
