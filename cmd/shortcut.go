package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	liqov1beta1 "github.com/liqotech/liqo/apis/core/v1beta1"
	networkingv1alpha1 "github.com/scal110/foreign_cluster_connector/api/v1beta1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/liqotech/liqo/pkg/liqo-controller-manager/networking/forge"
)

var (
	// nodeA and nodeB specify the two clusters to connect or disconnect
	nodeA string
	nodeB string

	// namespaceFlag sets the namespace for the CR
	namespaceFlag string

	// Networking parameters
	mtuFlag            int
	timeoutFlag        int
	waitFlag           bool

	// Server gateway configuration
	serverGatewayTypeFlag    string
	serverTemplateNameFlag   string
	serverTemplateNsFlag     string
	serverSvcTypeFlag        string
	serverSvcPortFlag        int

	// Client gateway configuration
	clientGatewayTypeFlag    string
	clientTemplateNameFlag   string
	clientTemplateNsFlag     string
)

func init() {
	// Register the top-level `shortcuts` command under rootCmd
	rootCmd.AddCommand(shortcutsCmd)

	// Subcommands for listing, creating, and deleting shortcuts
	shortcutsCmd.AddCommand(listShortcutsCmd)
	shortcutsCmd.AddCommand(createShortcutCmd)
	shortcutsCmd.AddCommand(deleteShortcutCmd)

	// Flags for `create` command
	createShortcutCmd.Flags().StringVarP(&nodeA, "node-a", "a", "", "Name of the first foreign cluster (required)")
	createShortcutCmd.Flags().StringVarP(&nodeB, "node-b", "b", "", "Name of the second foreign cluster (required)")
	createShortcutCmd.Flags().StringVar(&namespaceFlag, "namespace", "default", "Namespace in which to create the CR")
	createShortcutCmd.Flags().IntVar(&mtuFlag, "mtu", 1450, "MTU to configure for the tunnel interface")
	createShortcutCmd.Flags().IntVar(&timeoutFlag, "timeout", 120, "Connection timeout in seconds")
	createShortcutCmd.Flags().BoolVar(&waitFlag, "wait", true, "Wait until the connection is established")

	createShortcutCmd.Flags().StringVar(&serverGatewayTypeFlag, "server-gateway-type", forge.DefaultGwServerType, "Type of server gateway to create")
	createShortcutCmd.Flags().StringVar(&serverTemplateNameFlag, "server-template-name", forge.DefaultGwServerTemplateName, "Name of the WG server template")
	createShortcutCmd.Flags().StringVar(&serverTemplateNsFlag, "server-template-namespace", "liqo", "Namespace of the WG server template")
	createShortcutCmd.Flags().StringVar(&serverSvcTypeFlag, "server-service-type", "NodePort", "Kubernetes Service type for server gateway")
	createShortcutCmd.Flags().IntVar(&serverSvcPortFlag, "server-service-port", forge.DefaultGwServerPort, "Service port for server gateway")

	createShortcutCmd.Flags().StringVar(&clientGatewayTypeFlag, "client-gateway-type", forge.DefaultGwClientType, "Type of client gateway to create")
	createShortcutCmd.Flags().StringVar(&clientTemplateNameFlag, "client-template-name", forge.DefaultGwClientTemplateName, "Name of the WG client template")
	createShortcutCmd.Flags().StringVar(&clientTemplateNsFlag, "client-template-namespace", "liqo", "Namespace of the WG client template")

	// Mark required flags
	_ = createShortcutCmd.MarkFlagRequired("node-a")
	_ = createShortcutCmd.MarkFlagRequired("node-b")

	// Flags for `delete` command
	deleteShortcutCmd.Flags().StringVarP(&nodeA, "node-a", "a", "", "Name of the first foreign cluster (required)")
	deleteShortcutCmd.Flags().StringVarP(&nodeB, "node-b", "b", "", "Name of the second foreign cluster (required)")
	_ = deleteShortcutCmd.MarkFlagRequired("node-a")
	_ = deleteShortcutCmd.MarkFlagRequired("node-b")
}

// shortcutsCmd is the parent command for shortcut operations
var shortcutsCmd = &cobra.Command{
	Use:   "shortcuts",
	Short: "Manage ForeignClusterConnection resources in the current cluster",
	Long:  "Use subcommands to list, create, or delete cross-cluster shortcuts via the ForeignClusterConnection CRD.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// listShortcutsCmd lists all existing ForeignClusterConnection resources
var listShortcutsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured shortcuts",
	Run: func(cmd *cobra.Command, args []string) {
		if err := listShortcuts(); err != nil {
			fmt.Fprintf(os.Stderr, "Error listing shortcuts: %v\n", err)
			os.Exit(1)
		}
	},
}

// listShortcuts retrieves and prints all ForeignClusterConnection objects
func listShortcuts() error {
	ctx := context.Background()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get Kubernetes config: %w", err)
	}

	scheme := runtime.NewScheme()
	_ = networkingv1alpha1.AddToScheme(scheme) // register our CRD

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create API client: %w", err)
	}

	var list networkingv1alpha1.ForeignClusterConnectionList
	if err := cl.List(ctx, &list); err != nil {
		return fmt.Errorf("unable to list shortcuts: %w", err)
	}

	if len(list.Items) == 0 {
		fmt.Println("No shortcuts found.")
		return nil
	}

	// Print details for each shortcut
	for _, item := range list.Items {
		fmt.Printf("- %s <-> %s : connected=%t\n",
			item.Spec.ForeignClusterA, item.Spec.ForeignClusterB, item.Status.IsConnected)
	}

	return nil
}

// createShortcutCmd creates a new ForeignClusterConnection
var createShortcutCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a shortcut (ForeignClusterConnection) between two clusters",
	Run: func(cmd *cobra.Command, args []string) {
		if err := createShortcut(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating shortcut: %v\n", err)
			os.Exit(1)
		}
	},
}

// createShortcut constructs and submits a ForeignClusterConnection object
func createShortcut() error {
	ctx := context.Background()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get Kubernetes config: %w", err)
	}

	scheme := runtime.NewScheme()
	_ = liqov1beta1.AddToScheme(scheme)
	_ = networkingv1alpha1.AddToScheme(scheme)

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create API client: %w", err)
	}

	// Verify both clusters exist
	for _, name := range []string{nodeA, nodeB} {
		var fc liqov1beta1.ForeignCluster
		if err := cl.Get(ctx, client.ObjectKey{Name: name}, &fc); err != nil {
			return fmt.Errorf("foreign cluster %q not found: %w", name, err)
		}
	}

	// Check if a connection already exists
	var existing networkingv1alpha1.ForeignClusterConnectionList
	_ = cl.List(ctx, &existing)
	for _, item := range existing.Items {
		if (item.Spec.ForeignClusterA == nodeA && item.Spec.ForeignClusterB == nodeB) ||
			(item.Spec.ForeignClusterA == nodeB && item.Spec.ForeignClusterB == nodeA) {
			fmt.Printf("Connection already exists between %s and %s\n", nodeA, nodeB)
			return nil
		}
	}

	// Construct a new CR name from the two cluster names
	name := strings.ToLower(strings.ReplaceAll(fmt.Sprintf("%s-%s", nodeA, nodeB), "_", "-"))
	newConn := &networkingv1alpha1.ForeignClusterConnection{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespaceFlag},
		Spec: networkingv1alpha1.ForeignClusterConnectionSpec{
			ForeignClusterA: nodeA,
			ForeignClusterB: nodeB,
			Networking: networkingv1alpha1.NetworkingConfig{
				MTU:                      int32(mtuFlag),
				ServerGatewayType:        serverGatewayTypeFlag,
				ServerTemplateName:       serverTemplateNameFlag,
				ServerTemplateNamespace:  serverTemplateNsFlag,
				ServerServiceType:        serverSvcTypeFlag,
				ServerServicePort:        int32(serverSvcPortFlag),
				ClientGatewayType:        clientGatewayTypeFlag,
				ClientTemplateName:       clientTemplateNameFlag,
				ClientTemplateNamespace:  clientTemplateNsFlag,
				TimeoutSeconds:           int32(timeoutFlag),
				Wait:                     waitFlag,
			},
		},
	}

	// Create the CR resource
	if err := cl.Create(ctx, newConn); err != nil {
		return fmt.Errorf("failed to create ForeignClusterConnection: %w", err)
	}

	fmt.Printf("Created ForeignClusterConnection %s/%s successfully.\n", namespaceFlag, name)
	return nil
}

// deleteShortcutCmd removes an existing ForeignClusterConnection
var deleteShortcutCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an existing shortcut between two clusters",
	Run: func(cmd *cobra.Command, args []string) {
		if err := deleteShortcut(); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting shortcut: %v\n", err)
			os.Exit(1)
		}
	},
}

// deleteShortcut searches for and deletes the matching CR
func deleteShortcut() error {
	ctx := context.Background()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get Kubernetes config: %w", err)
	}

	scheme := runtime.NewScheme()
	_ = liqov1beta1.AddToScheme(scheme)
	_ = networkingv1alpha1.AddToScheme(scheme)

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create API client: %w", err)
	}

	// List all connections and find the one matching nodeA/nodeB
	var list networkingv1alpha1.ForeignClusterConnectionList
	if err := cl.List(ctx, &list); err != nil {
		return fmt.Errorf("unable to list connections: %w", err)
	}

	var target *networkingv1alpha1.ForeignClusterConnection
	for i := range list.Items {
		item := &list.Items[i]
		a, b := item.Spec.ForeignClusterA, item.Spec.ForeignClusterB
		if (a == nodeA && b == nodeB) || (a == nodeB && b == nodeA) {
			target = item
			break
		}
	}

	if target == nil {
		fmt.Printf("No shortcut found between %q and %q.\n", nodeA, nodeB)
		return nil
	}

	// Send delete request
	if err := cl.Delete(ctx, target); err != nil {
		return fmt.Errorf("failed to delete %q: %w", target.Name, err)
	}

	fmt.Printf("Deletion requested for ForeignClusterConnection %q\n", target.Name)
	return nil
}