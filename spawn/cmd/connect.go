package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/scttfrdmn/spore-host/pkg/i18n"
	"github.com/scttfrdmn/spore-host/spawn/pkg/aws"
	"github.com/spf13/cobra"
)

var (
	connectUser       string
	connectKey        string
	connectPort       int
	connectSessionMgr bool
)

var connectCmd = &cobra.Command{
	Use:     "connect <instance-id>",
	RunE:    runConnect,
	Aliases: []string{"ssh"},
	Args:    cobra.ExactArgs(1),
	// Short and Long will be set after i18n initialization
}

func init() {
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().StringVar(&connectUser, "user", "", "SSH username (default: ec2-user)")
	connectCmd.Flags().StringVar(&connectKey, "key", "", "SSH private key path")
	connectCmd.Flags().IntVar(&connectPort, "port", 22, "SSH port")
	connectCmd.Flags().BoolVar(&connectSessionMgr, "session-manager", false, "Use AWS Session Manager instead of SSH")

	// Register completion for instance ID argument
	connectCmd.ValidArgsFunction = completeInstanceID
}

func runConnect(cmd *cobra.Command, args []string) error {
	instanceIdentifier := args[0]
	ctx := context.Background()

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		return i18n.Te("error.aws_client_init", err)
	}

	// Resolve instance (by ID or name)
	instance, err := resolveInstance(ctx, client, instanceIdentifier)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "%s\n", i18n.Tf("spawn.connect.found_instance", map[string]interface{}{
		"Region": instance.Region,
		"State":  instance.State,
	}))

	// Check if instance is running
	if instance.State != "running" {
		return i18n.Te("spawn.connect.error.not_running", nil, map[string]interface{}{
			"State": instance.State,
		})
	}

	// Use Session Manager if requested or if no public IP
	if connectSessionMgr || instance.PublicIP == "" {
		return connectViaSessionManager(instance.InstanceID, instance.Region)
	}

	// Determine SSH user
	user := connectUser
	if user == "" {
		user = "ec2-user" // Default for Amazon Linux
	}

	// Determine SSH key
	keyPath := connectKey
	if keyPath == "" {
		// Try to find the key based on the instance key name
		keyPath, err = findSSHKey(instance.KeyName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s: %v\n", i18n.Symbol("warning"), i18n.Tf("spawn.connect.key_not_found", map[string]interface{}{
				"KeyName": instance.KeyName,
			}), err)
			fmt.Fprintf(os.Stderr, "%s\n\n", i18n.T("spawn.connect.fallback_session_manager"))
			return connectViaSessionManager(instance.InstanceID, instance.Region)
		}
	}

	// Build SSH command
	sshArgs := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", fmt.Sprintf("%d", connectPort),
		fmt.Sprintf("%s@%s", user, instance.PublicIP),
	}

	fmt.Fprintf(os.Stderr, "%s\n\n", i18n.Tf("spawn.connect.connecting_ssh", map[string]interface{}{
		"Command": "ssh " + strings.Join(sshArgs, " "),
	}))

	// Execute SSH
	sshCmd := exec.Command("ssh", sshArgs...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

func connectViaSessionManager(instanceID, region string) error {
	// Check if AWS CLI and Session Manager plugin are installed
	_, err := exec.LookPath("aws")
	if err != nil {
		return i18n.Te("spawn.connect.error.aws_cli_not_found", nil)
	}

	fmt.Fprintf(os.Stderr, "%s\n\n", i18n.T("spawn.connect.connecting_session_manager"))

	// Build AWS SSM start-session command
	ssmCmd := exec.Command("aws", "ssm", "start-session",
		"--target", instanceID,
		"--region", region,
	)

	ssmCmd.Stdin = os.Stdin
	ssmCmd.Stdout = os.Stdout
	ssmCmd.Stderr = os.Stderr

	err = ssmCmd.Run()
	if err != nil {
		return i18n.Te("spawn.connect.error.session_manager_failed", err)
	}

	return nil
}

func findSSHKey(keyName string) (string, error) {
	if keyName == "" {
		return "", i18n.Te("spawn.connect.error.no_key_name", nil)
	}

	// Common SSH key locations and naming patterns
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Try various common key names
	keyPatterns := []string{
		filepath.Join(sshDir, keyName),        // Exact name
		filepath.Join(sshDir, keyName+".pem"), // With .pem
		filepath.Join(sshDir, keyName+".key"), // With .key
		filepath.Join(sshDir, "id_rsa"),       // Default RSA key
		filepath.Join(sshDir, "id_ed25519"),   // Default Ed25519 key
		filepath.Join(sshDir, "id_ecdsa"),     // Default ECDSA key
	}

	for _, path := range keyPatterns {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", i18n.Te("spawn.connect.error.key_not_found_for_name", nil, map[string]interface{}{
		"KeyName": keyName,
	})
}
