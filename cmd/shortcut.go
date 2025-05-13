package cmd

import(
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

shortcutsCmd.AddCommand(deleteShortcutCmd)
// flags per delete
deleteShortcutCmd.Flags().StringVarP(&nodeA, "node-a", "a", "", "Nome del primo virtual node (required)")
deleteShortcutCmd.Flags().StringVarP(&nodeB, "node-b", "b", "", "Nome del secondo virtual node (required)")
deleteShortcutCmd.MarkFlagRequired("node-a")
deleteShortcutCmd.MarkFlagRequired("node-b")

}

// ---------------------------
// LIST SHORTCUTS
// ---------------------------
var shortcutsCmd = &cobra.Command{
Use:   "shortcuts",
Short: "List all ForeignClusterConnections in the current Kubernetes cluster",
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
Short: "Create a ForeignClusterConnection between two foreign clusters",
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
var existingList networkingv1alpha1.ForeignClusterConnectionList
if err := cl.List(ctx, &existingList); err != nil {
	return fmt.Errorf("impossible to list existing connections: %w", err)
}
for _, item := range existingList.Items {
	a := item.Spec.ForeignClusterA
	b := item.Spec.ForeignClusterB
	if (a == nodeA && b == nodeB) || (a == nodeB && b == nodeA) {
		if item.Status.IsConnected {
			fmt.Printf("Esiste già una connessione funzionante tra %s e %s\n", nodeA, nodeB)
			return nil
		}
		fmt.Printf("Esiste già una CRD tra %s e %s ma non è ancora connessa (phase=%s)\n", nodeA, nodeB, item.Status.Phase)
		return nil
	}
}

// Crea il nuovo ForeignClusterConnection
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
			MTU: mtuFlag,
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
	return fmt.Errorf("impossibile creare ForeignClusterConnection %q: %w", name, err)
}

fmt.Printf("ForeignClusterConnection %q creata con successo!\n", name)
return nil
}

var deleteShortcutCmd = &cobra.Command{
Use:   "delete",
Short: "Delete a ForeignClusterConnection between two foreign clusters",
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

// Lista tutte le connessioni e cerca quella corrispondente
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

// Elimina la CR: il controller gestirà la disconnessione via finalizer
if err := cl.Delete(ctx, toDelete); err != nil {
	return fmt.Errorf("failed to delete ForeignClusterConnection %q: %w", toDelete.Name, err)
}

fmt.Printf("Richiesta di cancellazione per ForeignClusterConnection %q inviata\n", toDelete.Name)
return nil
}