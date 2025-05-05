package cmd

import (
	"context"
	"fmt"
	"os"

	liqov1beta1 "github.com/liqotech/liqo/apis/core/v1beta1"
	networkingv1alpha1 "github.com/nates110/vnc-controller/api/v1beta1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ---------------------------
// LIST SHORTCUTS
// ---------------------------
var shortcutsCmd = &cobra.Command{
	Use:   "shortcuts",
	Short: "List all virtualNodeConnections in the current Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		if err := listShortcuts(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(shortcutsCmd) // aggiungi questo
}

func listShortcuts() error {
	ctx := context.Background()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := liqov1beta1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Liqo schema: %w", err)
	}

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	var shList networkingv1alpha1.VirtualNodeConnectionList
	if err := cl.List(ctx, &shList); err != nil {
		return fmt.Errorf("unable to list Shortcuts: %w", err)
	}

	if len(shList.Items) == 0 {
		fmt.Println("No Shortcuts found.")
		return nil
	}

	for _, sh := range shList.Items {
		fmt.Printf("- VirtualNodeA: %s\n  VirtualNodeB: %s\n  IsConnected: %t\n",
			sh.Spec.VirtualNodeA, sh.Spec.VirtualNodeB, sh.Status.IsConnected)
	}

	return nil
}