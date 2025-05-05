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
    nodeA string
    nodeB string
)

func init() {
    // existing
    rootCmd.AddCommand(shortcutsCmd)

    // aggiungi il sotto-comando "create"
    shortcutsCmd.AddCommand(createShortcutCmd)
    createShortcutCmd.Flags().StringVarP(&nodeA, "node-a", "a", "", "Nome del primo virtual node (required)")
    createShortcutCmd.Flags().StringVarP(&nodeB, "node-b", "b", "", "Nome del secondo virtual node (required)")
    createShortcutCmd.MarkFlagRequired("node-a")
    createShortcutCmd.MarkFlagRequired("node-b")
}

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

var createShortcutCmd = &cobra.Command{
    Use:   "create",
    Short: "Create a VirtualNodeConnection between two foreign clusters",
    Run: func(cmd *cobra.Command, args []string) {
        if err := createShortcut(); err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
    },
}

func createShortcut() error {
    ctx := context.Background()

    // Carica kubeconfig
    cfg, err := ctrl.GetConfig()
    if err != nil {
        return fmt.Errorf("unable to get kubeconfig: %w", err)
    }

    // Prepara lo scheme con Liqo e VNC
    scheme := runtime.NewScheme()
    if err := liqov1beta1.AddToScheme(scheme); err != nil {
        return fmt.Errorf("unable to add Liqo schema: %w", err)
    }
    if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
        return fmt.Errorf("unable to add VNC schema: %w", err)
    }

    // Crea il client
    cl, err := client.New(cfg, client.Options{Scheme: scheme})
    if err != nil {
        return fmt.Errorf("unable to create client: %w", err)
    }

    // Controlla che esistano i ForeignCluster
    for _, name := range []string{nodeA, nodeB} {
        var fc liqov1beta1.ForeignCluster
        if err := cl.Get(ctx, client.ObjectKey{Name: name}, &fc); err != nil {
            return fmt.Errorf("foreign cluster %q non trovato: %w", name, err)
        }
    }

    // Verifica se esiste già una connessione A↔B o B↔A
    var existingList networkingv1alpha1.VirtualNodeConnectionList
    if err := cl.List(ctx, &existingList); err != nil {
        return fmt.Errorf("impossible to list existing connections: %w", err)
    }
    for _, item := range existingList.Items {
        a := item.Spec.VirtualNodeA
        b := item.Spec.VirtualNodeB
        if (a == nodeA && b == nodeB) || (a == nodeB && b == nodeA) {
            if item.Status.IsConnected {
                fmt.Printf("Esiste già una connessione funzionante tra %s e %s\n", nodeA, nodeB)
                return nil
            }
            fmt.Printf("Esiste già una CRD tra %s e %s ma non è ancora connessa (phase=%s)\n", nodeA, nodeB, item.Status.Phase)
            return nil
        }
    }

    // Crea il nuovo VirtualNodeConnection
    name := fmt.Sprintf("%s-%s", nodeA, nodeB)
    // assicurati che il nome sia valide: minuscole e senza spazi
    name = strings.ToLower(strings.ReplaceAll(name, "_", "-"))

    vnc := &networkingv1alpha1.VirtualNodeConnection{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: "default", // o un flag se vuoi parametrizzare
        },
        Spec: networkingv1alpha1.VirtualNodeConnectionSpec{
            VirtualNodeA: nodeA,
            // KubeconfigA e B vengono recuperati dal controller runtime
            KubeconfigA: "",
            VirtualNodeB: nodeB,
            KubeconfigB: "",
        },
    }

    if err := cl.Create(ctx, vnc); err != nil {
        return fmt.Errorf("impossibile creare VirtualNodeConnection %q: %w", name, err)
    }

    fmt.Printf("VirtualNodeConnection %q creata con successo!\n", name)
    return nil
}
