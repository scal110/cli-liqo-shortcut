
package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	// repoURL is the base GitHub URL for the ForeignClusterConnector project.
	repoURL = "https://github.com/scal110/foreign_cluster_connector"
	// kustomizePath points to the kustomize directory with controller manifests.
	kustomizePath = repoURL + "/config/default"
)

// managerCmd is an intermediate command grouping deploy/remove under "manager"
var managerCmd = &cobra.Command{
	Use:   "manager",
	Short: "Manage ForeignClusterConnector controller",
	Long: `Group commands to install or uninstall the controller and its RBAC across
all discovered foreign clusters:
  • manager deploy → install controller + RBAC on all foreign clusters
  • manager remove → uninstall controller + remove RBAC from all foreign clusters`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand is provided, display help
		return cmd.Help()
	},
}

var deployCmd = &cobra.Command{
	Use:   "setup",
	Short: "Deploy the ForeignClusterConnector controller in the cluster",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔧 Deploying controller...")

		if err := runKustomizeCommand("apply"); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Setup failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Controller deployed successfully.")
	},
}

var undeployCmd = &cobra.Command{
	Use:   "undeploy",
	Short: "Remove the ForeignClusterConnector controller from the cluster",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🧹 Removing controller...")

		if err := runKustomizeCommand("delete"); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Undeploy failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Controller resources deleted.")
	},
}

func runKustomizeCommand(action string) error {
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "kubectl", action, "-k", kustomizePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("📦 Running: kubectl %s -k %s\n", action, kustomizePath)
	return cmd.Run()
}

func init() {
	// Attach manager under the main rootCmd
	rootCmd.AddCommand(managerCmd)
	// Register deploy & remove commands under manager
	managerCmd.AddCommand(deployCmd, removeCmd)
}