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
)

var (
	nodeA               string
	nodeB               string
	namespaceFlag       string
	mtuFlag             int
	disableSharingFlag  bool
	timeoutFlag         int
	waitFlag            bool
	serverGatewayTypeFlag    string
	serverTemplateNameFlag   string
	serverTemplateNsFlag     string
	serverSvcTypeFlag        string
	serverSvcPortFlag        int
	clientGatewayTypeFlag    string
	clientTemplateNameFlag   string
	clientTemplateNsFlag     string
)

func init() {
	rootCmd.AddCommand(shortcutsCmd)

	shortcutsCmd.AddCommand(createShortcutCmd)
	createShortcutCmd.Flags().StringVarP(&nodeA, "node-a", "a", "", "Nome del primo virtual node (required)")
	createShortcutCmd.Flags().StringVarP(&nodeB, "node-b", "b", "", "Nome del secondo virtual node (required)")
	createShortcutCmd.Flags().StringVar(&namespaceFlag, "namespace", "default", "Namespace della CR")
	createShortcutCmd.Flags().IntVar(&mtuFlag, "mtu", 1450, "MTU da utilizzare")
	createShortcutCmd.Flags().BoolVar(&disableSharingFlag, "disable-sharing", false, "Disabilita la condivisione chiavi")
	createShortcutCmd.Flags().IntVar(&timeoutFlag, "timeout", 120, "Timeout della connessione in secondi")
	createShortcutCmd.Flags().BoolVar(&waitFlag, "wait", true, "Attendi il completamento della connessione")
	createShortcutCmd.Flags().StringVar(&serverGatewayTypeFlag, "server-gateway-type", "", "Tipo gateway server")
	createShortcutCmd.Flags().StringVar(&serverTemplateNameFlag, "server-template-name", "", "Nome template server")
	createShortcutCmd.Flags().StringVar(&serverTemplateNsFlag, "server-template-namespace", "liqo", "Namespace del template server")
	createShortcutCmd.Flags().StringVar(&serverSvcTypeFlag, "server-service-type", "", "Tipo di service server")
	createShortcutCmd.Flags().IntVar(&serverSvcPortFlag, "server-service-port", 0, "Porta del service server")
	createShortcutCmd.Flags().StringVar(&clientGatewayTypeFlag, "client-gateway-type", "", "Tipo gateway client")
	createShortcutCmd.Flags().StringVar(&clientTemplateNameFlag, "client-template-name", "", "Nome template client")
	createShortcutCmd.Flags().StringVar(&clientTemplateNsFlag, "client-template-namespace", "liqo", "Namespace del template client")
	createShortcutCmd.MarkFlagRequired("node-a")
	createShortcutCmd.MarkFlagRequired("node-b")

	shortcutsCmd.AddCommand(deleteShortcutCmd)
	deleteShortcutCmd.Flags().StringVarP(&nodeA, "node-a", "a", "", "Nome del primo virtual node (required)")
	deleteShortcutCmd.Flags().StringVarP(&nodeB, "node-b", "b", "", "Nome del secondo virtual node (required)")
	deleteShortcutCmd.MarkFlagRequired("node-a")
	deleteShortcutCmd.MarkFlagRequired("node-b")
}

var shortcutsCmd = &cobra.Command{
	Use:   "shortcuts",
	Short: "Gestisci ForeignClusterConnections nel cluster Kubernetes corrente",
	Run: func(cmd *cobra.Command, args []string) {
		if err := listShortcuts(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func listShortcuts() error {
	ctx := context.Background()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Liqo schema: %w", err)
	}

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	var shList networkingv1alpha1.ForeignClusterConnectionList
	if err := cl.List(ctx, &shList); err != nil {
		return fmt.Errorf("unable to list Shortcuts: %w", err)
	}

	if len(shList.Items) == 0 {
		fmt.Println("No Shortcuts found.")
		return nil
	}

	for _, sh := range shList.Items {
		fmt.Printf("- ForeignClusterA: %s\n  ForeignClusterB: %s\n  IsConnected: %t\n",
			sh.Spec.ForeignClusterA, sh.Spec.ForeignClusterB, sh.Status.IsConnected)
	}

	return nil
}

var createShortcutCmd = &cobra.Command{
	Use:   "create",
	Short: "Crea una ForeignClusterConnection tra due foreign clusters",
	Run: func(cmd *cobra.Command, args []string) {
		if err := createShortcut(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func createShortcut() error {
	ctx := context.Background()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := liqov1beta1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Liqo schema: %w", err)
	}
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add VNC schema: %w", err)
	}

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	for _, name := range []string{nodeA, nodeB} {
		var fc liqov1beta1.ForeignCluster
		if err := cl.Get(ctx, client.ObjectKey{Name: name}, &fc); err != nil {
			return fmt.Errorf("foreign cluster %q non trovato: %w", name, err)
		}
	}

	var existing networkingv1alpha1.ForeignClusterConnectionList
	if err := cl.List(ctx, &existing); err != nil {
		return err
	}
	for _, item := range existing.Items {
		if (item.Spec.ForeignClusterA == nodeA && item.Spec.ForeignClusterB == nodeB) ||
			(item.Spec.ForeignClusterA == nodeB && item.Spec.ForeignClusterB == nodeA) {
			fmt.Printf("Connessione già esistente tra %s e %s\n", nodeA, nodeB)
			return nil
		}
	}

	name := strings.ToLower(strings.ReplaceAll(fmt.Sprintf("%s-%s", nodeA, nodeB), "_", "-"))
	newFcc := &networkingv1alpha1.ForeignClusterConnection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceFlag,
		},
		Spec: networkingv1alpha1.ForeignClusterConnectionSpec{
			ForeignClusterA: nodeA,
			ForeignClusterB: nodeB,
			Networking: networkingv1alpha1.NetworkingConfig{
				MTU: int32(mtuFlag),
				DisableSharingKeys: disableSharingFlag,
				ServerGatewayType: serverGatewayTypeFlag,
				ServerTemplateName: serverTemplateNameFlag,
				ServerTemplateNamespace: serverTemplateNsFlag,
				ServerServiceType: serverSvcTypeFlag,
				ServerServicePort: int32(serverSvcPortFlag),
				ClientGatewayType: clientGatewayTypeFlag,
				ClientTemplateName: clientTemplateNameFlag,
				ClientTemplateNamespace: clientTemplateNsFlag,
				TimeoutSeconds: int32(timeoutFlag),
				Wait: waitFlag,
			},
		},
	}

	if err := cl.Create(ctx, newFcc); err != nil {
		return fmt.Errorf("errore nella creazione: %w", err)
	}

	fmt.Printf("CR %s/%s creata correttamente.\n", namespaceFlag, name)
	return nil
}

var deleteShortcutCmd = &cobra.Command{
	Use:   "delete",
	Short: "Elimina una ForeignClusterConnection esistente",
	Run: func(cmd *cobra.Command, args []string) {
		if err := deleteShortcut(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func deleteShortcut() error {
	ctx := context.Background()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := liqov1beta1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Liqo schema: %w", err)
	}
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add VNC schema: %w", err)
	}

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	var list networkingv1alpha1.ForeignClusterConnectionList
	if err := cl.List(ctx, &list); err != nil {
		return fmt.Errorf("unable to list ForeignClusterConnections: %w", err)
	}

	var toDelete *networkingv1alpha1.ForeignClusterConnection
	for i, item := range list.Items {
		a, b := item.Spec.ForeignClusterA, item.Spec.ForeignClusterB
		if (a == nodeA && b == nodeB) || (a == nodeB && b == nodeA) {
			toDelete = &list.Items[i]
			break
		}
	}

	if toDelete == nil {
		fmt.Printf("Nessuna ForeignClusterConnection trovata tra %q e %q\n", nodeA, nodeB)
		return nil
	}

	if err := cl.Delete(ctx, toDelete); err != nil {
		return fmt.Errorf("failed to delete ForeignClusterConnection %q: %w", toDelete.Name, err)
	}

	fmt.Printf("Richiesta di cancellazione per ForeignClusterConnection %q inviata\n", toDelete.Name)
	return nil
}
