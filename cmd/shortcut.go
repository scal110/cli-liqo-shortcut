package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	liqov1beta1 "github.com/liqotech/liqo/apis/core/v1beta1"
	networkingv1alpha1 "github.com/scal110/foreign_cluster_connector/api/v1beta1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/liqotech/liqo/pkg/liqo-controller-manager/networking/forge"
)

var (
	nodeA               string
	nodeB               string
	mtuFlag             int
	timeoutFlag         int
	waitFlag            bool
	disableSharingFlag  bool
	namespaceFlag       string

	serverSvcTypeFlag        corev1.ServiceType
	serverGatewayTypeFlag    string
	serverTemplateNameFlag   string
	serverTemplateNsFlag     string
	serverSvcPortFlag        int
	serverSvcType			 string

	clientGatewayTypeFlag    string
	clientTemplateNameFlag   string
	clientTemplateNsFlag     string
)

func init() {
	// Comando principale
	rootCmd.AddCommand(shortcutsCmd)

	// --- CREATE ---
	shortcutsCmd.AddCommand(createShortcutCmd)
	createShortcutCmd.Flags().StringVarP(&nodeA, "node-a", "a", "", "Nome del primo foreign cluster (required)")
	createShortcutCmd.Flags().StringVarP(&nodeB, "node-b", "b", "", "Nome del secondo foreign cluster (required)")
	createShortcutCmd.Flags().StringVar(&namespaceFlag, "namespace", "default", "Namespace della CR")

	createShortcutCmd.MarkFlagRequired("node-a")
	createShortcutCmd.MarkFlagRequired("node-b")

	createShortcutCmd.Flags().IntVar(&mtuFlag, "mtu", forge.DefaultMTU, "MTU per la connessione")
	createShortcutCmd.Flags().IntVar(&timeoutFlag, "timeout", 120, "Timeout in secondi")
	createShortcutCmd.Flags().BoolVar(&waitFlag, "wait", true, "Attendi che la connessione sia stabilita")
	createShortcutCmd.Flags().BoolVar(&disableSharingFlag, "disable-sharing", false, "Disabilita lo sharing delle chiavi")

	createShortcutCmd.Flags().StringVar(&serverGatewayTypeFlag, "server-gateway-type", forge.DefaultGwServerType, "Tipo di Gateway Server")
	createShortcutCmd.Flags().StringVar(&serverTemplateNameFlag, "server-template-name", forge.DefaultGwServerTemplateName, "Nome del template Server")
	createShortcutCmd.Flags().StringVar(&serverTemplateNsFlag, "server-template-namespace", "liqo", "Namespace del template Server")
	createShortcutCmd.Flags().IntVar(&serverSvcPortFlag, "gw-server-service-port", forge.DefaultGwServerPort, "Porta del service Server")
	createShortcutCmd.Flags().Var(&serverSvcType, "gw-server-service-type",forge.DefaultGwServerServiceType, "Tipo di service (ClusterIP|NodePort|LoadBalancer)")

	createShortcutCmd.Flags().StringVar(&clientGatewayTypeFlag, "client-gateway-type", forge.DefaultGwClientType, "Tipo di Gateway Client")
	createShortcutCmd.Flags().StringVar(&clientTemplateNameFlag, "client-template-name", forge.DefaultGwClientTemplateName, "Nome del template Client")
	createShortcutCmd.Flags().StringVar(&clientTemplateNsFlag, "client-template-namespace", "liqo", "Namespace del template Client")

	// --- DELETE ---
	shortcutsCmd.AddCommand(deleteShortcutCmd)
	deleteShortcutCmd.Flags().StringVarP(&nodeA, "node-a", "a", "", "Nome del primo foreign cluster (required)")
	deleteShortcutCmd.Flags().StringVarP(&nodeB, "node-b", "b", "", "Nome del secondo foreign cluster (required)")
	deleteShortcutCmd.Flags().StringVar(&namespaceFlag, "namespace", "default", "Namespace della CR")
	deleteShortcutCmd.MarkFlagRequired("node-a")
	deleteShortcutCmd.MarkFlagRequired("node-b")
}

// ---------------------------
// COMANDO PRINCIPALE
// ---------------------------
var shortcutsCmd = &cobra.Command{
	Use:   "shortcuts",
	Short: "Gestisci ForeignClusterConnection tra cluster remoti",
	Run: func(cmd *cobra.Command, args []string) {
		if err := listShortcuts(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// ---------------------------
// LIST SHORTCUTS
// ---------------------------
func listShortcuts() error {
	cl, err := getClientWithSchemes(liqov1beta1.AddToScheme, networkingv1alpha1.AddToScheme)
	if err != nil {
		return err
	}

	var shList networkingv1alpha1.ForeignClusterConnectionList
	if err := cl.List(context.TODO(), &shList); err != nil {
		return fmt.Errorf("unable to list connections: %w", err)
	}

	if len(shList.Items) == 0 {
		fmt.Println("No Shortcuts found.")
		return nil
	}

	for _, sh := range shList.Items {
		fmt.Printf("- ForeignClusterA: %s\n  ForeignClusterB: %s\n  IsConnected: %t (Phase: %s)\n",
			sh.Spec.ForeignClusterA, sh.Spec.ForeignClusterB, sh.Status.IsConnected, sh.Status.Phase)
	}
	return nil
}

// ---------------------------
// CREATE SHORTCUT
// ---------------------------
var createShortcutCmd = &cobra.Command{
	Use:   "create",
	Short: "Crea una ForeignClusterConnection tra due cluster remoti",
	Run: func(cmd *cobra.Command, args []string) {
		if err := createShortcut(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func createShortcut() error {
	cl, err := getClientWithSchemes(liqov1beta1.AddToScheme, networkingv1alpha1.AddToScheme)
	if err != nil {
		return err
	}

	// Verifica l'esistenza dei due ForeignCluster
	for _, name := range []string{nodeA, nodeB} {
		var fc liqov1beta1.ForeignCluster
		if err := cl.Get(context.TODO(), client.ObjectKey{Name: name}, &fc); err != nil {
			return fmt.Errorf("foreign cluster %q non trovato: %w", name, err)
		}
	}

	// Verifica se esiste già
	var existing networkingv1alpha1.ForeignClusterConnectionList
	if err := cl.List(context.TODO(), &existing); err != nil {
		return err
	}
	for _, item := range existing.Items {
		if (item.Spec.ForeignClusterA == nodeA && item.Spec.ForeignClusterB == nodeB) ||
			(item.Spec.ForeignClusterA == nodeB && item.Spec.ForeignClusterB == nodeA) {
			fmt.Printf("Connessione già esistente tra %s e %s (Phase: %s)\n", nodeA, nodeB, item.Status.Phase)
			return nil
		}
	}

	// Crea nuova CR
	name := strings.ToLower(strings.ReplaceAll(fmt.Sprintf("%s-%s", nodeA, nodeB), "_", "-"))
	newVNC := &networkingv1alpha1.ForeignClusterConnection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceFlag,
		},
		Spec: networkingv1alpha1.ForeignClusterConnectionSpec{
			ForeignClusterA: nodeA,
			ForeignClusterB: nodeB,
		},
	}

	if err := cl.Create(context.TODO(), newVNC); err != nil {
		return fmt.Errorf("errore nella creazione: %w", err)
	}

	fmt.Printf("Creato: %s/%s\n", namespaceFlag, name)

	if waitFlag {
		fmt.Print("Attesa connessione ")
		for i := 0; i < timeoutFlag; i++ {
			var updated networkingv1alpha1.ForeignClusterConnection
			if err := cl.Get(context.TODO(), client.ObjectKey{Namespace: namespaceFlag, Name: name}, &updated); err != nil {
				return err
			}
			if updated.Status.IsConnected {
				fmt.Println("✓ Connesso!")
				return nil
			}
			fmt.Print(".")
			time.Sleep(1 * time.Second)
		}
		fmt.Println("\n⚠️ Timeout scaduto, connessione non ancora stabilita.")
	}

	return nil
}

// ---------------------------
// DELETE SHORTCUT
// ---------------------------
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
	cl, err := getClientWithSchemes(liqov1beta1.AddToScheme, networkingv1alpha1.AddToScheme)
	if err != nil {
		return err
	}

	var list networkingv1alpha1.ForeignClusterConnectionList
	if err := cl.List(context.TODO(), &list); err != nil {
		return err
	}

	for _, item := range list.Items {
		if (item.Spec.ForeignClusterA == nodeA && item.Spec.ForeignClusterB == nodeB) ||
			(item.Spec.ForeignClusterA == nodeB && item.Spec.ForeignClusterB == nodeA) {
			if err := cl.Delete(context.TODO(), &item); err != nil {
				return fmt.Errorf("errore nella cancellazione: %w", err)
			}
			fmt.Printf("Eliminato: %s/%s\n", item.Namespace, item.Name)
			return nil
		}
	}
	fmt.Println("Nessuna connessione trovata.")
	return nil
}

// ---------------------------
// UTILITY: client init
// ---------------------------
func getClientWithSchemes(schemes ...func(*runtime.Scheme) error) (client.Client, error) {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get kubeconfig: %w", err)
	}
	scheme := runtime.NewScheme()
	for _, add := range schemes {
		if err := add(scheme); err != nil {
			return nil, err
		}
	}
	return client.New(cfg, client.Options{Scheme: scheme})
}
