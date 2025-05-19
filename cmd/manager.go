
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

// deployCmd installs the controller on the primary cluster and sets up RBAC
// on every foreign cluster detected in the primary cluster.
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Install controller + set up RBAC on all foreign clusters",
	Run: func(cmd *cobra.Command, args []string) {
		// 1) Identify current user
		user, err := currentKubectlUser()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to get current user: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("ℹ️  Current Kubernetes user: %s\n\n", user)

		// 2) Deploy controller in primary cluster
		fmt.Println("🔧 Deploying controller to primary cluster...")
		if err := runKubectl("apply", "-k", kustomizePath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Controller deploy failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Controller deployed.\n")

		// 3) Discover all foreign clusters
		fcs, err := listForeignClusters()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Could not list foreign clusters: %v\n", err)
			os.Exit(1)
		}
		if len(fcs) == 0 {
			fmt.Println("⚠️  No foreign clusters detected; skipping RBAC setup.")
			return
		}

		// 4) Apply RBAC on each
		for _, fc := range fcs {
			fmt.Printf("🌍 Applying RBAC on foreign '%s'...\n", fc)

			kubeconfig, err := fetchForeignKubeconfig(fc)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Fetching kubeconfig for '%s' failed: %v\n", fc, err)
				os.Exit(1)
			}
			defer os.Remove(kubeconfig)

			rbac := rbacYAML(user)
			kc := exec.CommandContext(
				context.Background(),
				"kubectl", "--kubeconfig", kubeconfig, "apply", "-f", "-",
			)
			kc.Stdin = strings.NewReader(rbac)
			kc.Stdout = os.Stdout
			kc.Stderr = os.Stderr

			fmt.Printf("📦 kubectl --kubeconfig=%s apply -f -\n", kubeconfig)
			if err := kc.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ RBAC apply failed on '%s': %v\n", fc, err)
				os.Exit(1)
			}
			fmt.Printf("✅ RBAC applied on '%s'.\n\n", fc)
		}
	},
}

// removeCmd uninstalls the controller and removes RBAC from all foreign clusters.
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Uninstall controller + remove RBAC from all foreign clusters",
	Run: func(cmd *cobra.Command, args []string) {
		// 1) Identify current user
		user, err := currentKubectlUser()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to get current user: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("ℹ️  Current Kubernetes user: %s\n\n", user)

		// 2) Remove controller from primary cluster
		fmt.Println("🧹 Removing controller from primary cluster...")
		if err := runKubectl("delete", "-k", kustomizePath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Controller removal failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Controller resources deleted.\n")

		// 3) Discover remote clusters
		fcs, err := listForeignClusters()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Could not list foreign clusters: %v\n", err)
			os.Exit(1)
		}
		if len(fcs) == 0 {
			fmt.Println("⚠️  No foreign clusters detected; skipping RBAC cleanup.")
			return
		}

		// 4) Delete RBAC on each
		for _, fc := range fcs {
			fmt.Printf("🌍 Removing RBAC on foreign '%s'...\n", fc)

			kubeconfig, err := fetchForeignKubeconfig(fc)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Fetching kubeconfig for '%s' failed: %v\n", fc, err)
				os.Exit(1)
			}
			defer os.Remove(kubeconfig)

			rbac := rbacYAML(user)
			kc := exec.CommandContext(
				context.Background(),
				"kubectl", "--kubeconfig", kubeconfig, "delete", "-f", "-",
			)
			kc.Stdin = strings.NewReader(rbac)
			kc.Stdout = os.Stdout
			kc.Stderr = os.Stderr

			fmt.Printf("📦 kubectl --kubeconfig=%s delete -f -\n", kubeconfig)
			if err := kc.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ RBAC delete failed on '%s': %v\n", fc, err)
				os.Exit(1)
			}
			fmt.Printf("✅ RBAC removed on '%s'.\n\n", fc)
		}
	},
}

func init() {
	// Attach managerCmd under rootCmd (defined in cmd/root.go)
	rootCmd.AddCommand(managerCmd)
	// Register deploy & remove under manager
	managerCmd.AddCommand(deployCmd, removeCmd)
}

// listForeignClusters discovers all ForeignCluster CRs in the primary cluster
// by querying the API and returning their names.
func listForeignClusters() ([]string, error) {
	out, err := exec.Command("kubectl", "get", "foreignclusters",
		"-o", "jsonpath={.items[*].metadata.name}").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error listing foreignclusters: %v (%s)", err, out)
	}
	fields := strings.Fields(string(out))
	return fields, nil
}

// currentKubectlUser returns the user name from the active kubeconfig context.
func currentKubectlUser() (string, error) {
	out, err := exec.Command("kubectl", "config", "view", "--minify",
		"-o", "jsonpath={.users[0].name}").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cannot retrieve kubectl user: %v (%s)", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// fetchForeignKubeconfig retrieves the kubeconfig secret for a given foreign cluster
// from the primary cluster, decodes it, and writes it to a temp file.
func fetchForeignKubeconfig(clusterName string) (string, error) {
	ns := fmt.Sprintf("liqo-tenant-%s", clusterName)
	secret := fmt.Sprintf("kubeconfig-controlplane-%s", clusterName)

	out, err := exec.Command("kubectl", "get", "secret", secret,
		"-n", ns,
		"-o", "jsonpath={.data.kubeconfig}").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error fetching secret %s/%s: %v (%s)", ns, secret, err, out)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(out))
	if err != nil {
		return "", fmt.Errorf("error decoding kubeconfig for %s: %v", clusterName, err)
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("kubeconfig-%s.yaml", clusterName))
	if err := os.WriteFile(tmp, decoded, 0600); err != nil {
		return "", fmt.Errorf("error writing kubeconfig file %s: %v", tmp, err)
	}
	return tmp, nil
}

// rbacYAML returns the RBAC manifests (ClusterRole + ClusterRoleBinding)
// with the supplied user as the binding subject.
func rbacYAML(user string) string {
	return fmt.Sprintf(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: liqo-gatewayclients-reader
rules:
- apiGroups: ["ipam.liqo.io","networking.liqo.io","apps",""]
  resources: ["namespaces","internalfabrics","publickeies","configmaps","wggatewayservertemplates","wggatewayclienttemplates","secrets","connections","networks","services","deployments","gatewayclients","gatewayservers","configurations"]
  verbs: ["get","list","watch","delete","create","update","patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: liqo-gatewayclients-reader-binding
subjects:
- kind: User
  name: %s
roleRef:
  kind: ClusterRole
  name: liqo-gatewayclients-reader
  apiGroup: rbac.authorization.k8s.io
`, user)
}

// runKubectl executes a kubectl command and streams its output.
func runKubectl(args ...string) error {
	cmd := exec.CommandContext(context.Background(), "kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("📦 kubectl %s\n", strings.Join(args, " "))
	return cmd.Run()
}
