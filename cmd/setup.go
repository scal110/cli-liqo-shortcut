package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	repoURL  = "https://github.com/scal110/foreign_cluster_connector"
	kustomizePath = repoURL + "/config/default"
)

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(undeployCmd)
}

var setupCmd = &cobra.Command{
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
