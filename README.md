# CLI Liqo Shortcut

This CLI tool provides simple commands to deploy and remove the **ForeignClusterConnector** controller in a Liqo multi‑cluster environment and to manage direct connections between foreign clusters ("shortcuts") via the `ForeignClusterConnection` CR.

**Controller project:** [https://github.com/scal110/foreign\_cluster\_connector.git](https://github.com/scal110/foreign_cluster_connector.git)

---

## Prerequisites

* A Kubernetes setup using the Liqo **replicated‑deployments** example
* **Cilium** as the CNI plugin (instead of the default Kindnet), due to known `liqoctl disconnect/reset` issues with Kindnet
* `kubectl` configured to access your **primary** cluster

**Temporary RBAC workaround:**

> The controller needs elevated permissions in each foreign cluster. Apply:
>
> ```bash
> kubectl apply -f clusterrole.yaml \
>   --kubeconfig /path/to/foreign/kubeconfig
> ```
>
>⚠️ This binding is for testing only. A proper ServiceAccount with the necessary ClusterRole and ClusterRoleBinding should be configured for production use.
> 
>⚠️ If a non-default setup is used, ensure that the `subjects` field in the `ClusterRoleBinding` is updated to reference the correct ServiceAccount, User, or Group of the main cluster.
---

## Installation

Clone and build the CLI:

```bash
git clone https://github.com/scal110/cli-liqo-shortcut.git
cd cli-liqo-shortcut
# Build a binary named "liqoshortcut"
go build -o liqoshortcut main.go
sudo mv liqoshortcut /usr/local/bin/
```

Confirm it’s on your PATH:

```bash
liqoshortcut --help
```

---

## Usage Overview

All commands are under two primary namespaces:

| Namespace     | Purpose                              | Common Flags                               |
| ------------- | ------------------------------------ | ------------------------------------------ |
| **manager**   | Controller lifecycle (deploy/remove) | *none*                                     |
| **shortcuts** | Manage cluster shortcuts (CRs)       | `-a`, `--cluster-a`<br>`-b`, `--cluster-b` |

Each supports `--help`:

```bash
liqoshortcut manager --help
liqoshortcut shortcuts --help
```

---

## Manager Commands

Install or uninstall the controller in your primary cluster:

| Command                       | Description                                   | Example                       |
| ----------------------------- | --------------------------------------------- | ----------------------------- |
| `liqoshortcut manager deploy` | Deploy the ForeignClusterConnector controller | `liqoshortcut manager deploy` |
| `liqoshortcut manager remove` | Remove the controller resources               | `liqoshortcut manager remove` |


---

## Shortcuts Commands

Create, list, and delete **ForeignClusterConnection** CRs to establish network shortcuts.

| Command                                                     | Flags                                      | Description                            | Example                                                                  |
| ----------------------------------------------------------- | ------------------------------------------ | -------------------------------------- | ------------------------------------------------------------------------ |
| `liqoshortcut shortcuts list`                               | *none*                                     | List all existing shortcuts            | `liqoshortcut shortcuts list`                                            |
| `liqoshortcut shortcuts create -a <clusterA> -b <clusterB>` | `-a`, `--cluster-a`<br>`-b`, `--cluster-b` | Create a shortcut between two clusters | `liqoshortcut shortcuts create -a europe-rome-edge -b europe-milan-edge` |
| `liqoshortcut shortcuts delete -a <clusterA> -b <clusterB>` | same as create                             | Delete the specified shortcut          | `liqoshortcut shortcuts delete -a europe-rome-edge -b europe-milan-edge` |

Example workflow:

```bash
# Create a network shortcut between "europe-rome-edge" and "europe-milan-edge"
liqoshortcut shortcuts create -a europe-rome-edge -b europe-milan-edge

# Verify:
liqoshortcut shortcuts list

# Tear down:
liqoshortcut shortcuts delete -a europe-rome-edge -b europe-milan-edge
```

---

## Troubleshooting

* **Forbidden** errors when creating shortcuts: apply `clusterrole.yaml` in each foreign cluster (see Prerequisites).
* Use `--help` on any command to inspect available flags and syntax.

---

Happy multi‑cluster networking with Liqo! 🚀
