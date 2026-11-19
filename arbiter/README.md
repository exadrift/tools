# arbiter

A guided, GitOps-oriented tool for Talos cluster management.

This tool streamlines the management of single-node Talos clusters with a focus on reproducibility, ease of management, and secure secret handling.

## Features

- **Single-node Talos cluster management**
- **Secure secret management with GitOps integration**
- **Automated ArgoCD deployment**
- **Git-friendly cluster configuration**
- **Easy-to-use CLI with guided workflows**

## Installation

### Using the installation script (recommended)
```bash
# Binary will be installed to ~/.local/bin
# To install elsewhere, set BIN_DIR to the desired location
sh -c "curl https://raw.githubusercontent.com/exadrift/tools/refs/heads/main/install.sh | sh -s -- arbiter"
```

### Manual installation
Build from source:
```bash
go build -o arbiter .
```

## Usage

### Prerequisites
- A Talos cluster (single-node recommended)
- Git repository for configuration storage
- `talosctl` installed (will be automatically installed if needed)

### Basic Workflow

#### 1. Prepare Talos Image
Visit the [Talos Image Factory](https://factory.talos.dev/) to create a bootable image for your target hardware.

Before proceeding, boot the target machine from the Talos installer image to put it in "maintenance mode".

#### 2. Initialize a Project
Create a blank git repository and clone it locally. This will store all your cluster configuration.

```bash
# From the directory of the git repository root
arbiter init
```

This will walk you through the configuration process (machine IP, etc.). At the end of this process, basic configuration will be established at `~/.arbiter/<project-name>`. You will be prompted to back up the `config.yaml` which will be generated there. This file contains the configuration but most importantly the key material which will be used to manage secrets stored in the GitHub repository. If you lose this file, you won't be able to encrypt/decrypt those secrets anymore, which means you won't be able to contact the Talos cluster.

#### 3. Bootstrap Talos
```bash
# If you're still in the git repo, or if not, supply the project name as an argument
arbiter bootstrap
```

You will be prompted for the install disk on the target machine, then the process will begin. The machine may reboot during this process, so make sure your install media is not the first to boot by default, as this could cause installation to hang indefinitely.

This will automatically install ArgoCD into the cluster and create additional manifests for an app-in-apps pattern in your Git repository.

#### 4. Test kubectl
At the end of the bootstrap process, `arbiter` will configure `kubectl`. You can now interrogate the new cluster via `kubectl`. You may have to select the new context first, which will be named the same as your arbiter project.

#### 5. Post-Configuration
```bash
arbiter post-configure
```

This stage gives a guided process for adding additional resources such as disks.

#### 6. Applying Future Changes
```bash
arbiter apply
```

This will apply Talos machine and/or Kubernetes patches which were prepared during the post-configuration stage, or manually added to the repo's `config/patches` section. Patches should be alphabetically ordered for installation preference.

- `/machine` must contain only Talos machine configuration
- `/kubernetes` must contain only Kubernetes assets which are not to be managed by ArgoCD, or other tools

#### 7. Secret Management
`arbiter` allows you to store secrets in Git, and while secrets are not applied via GitOps, they are applied just as normal Kubernetes patches would be, using `arbiter apply`.

Arbiter secrets have two components:
1. The secret definition in arbiter
2. The linkage to a Kubernetes secret

##### Defining a Secret in Arbiter
```bash
arbiter secrets set <key-name>
```
This allows you to set a secret from either a `--path` or will prompt for input from stdin.

##### Listing Defined Secrets in Arbiter
```bash
arbiter secrets list
```

##### Creating a Kubernetes Secret that References Arbiter
```bash
arbiter secrets apply <kube-secret-name> --namespace <ns>
```
This will guide you through the creation of a named Kubernetes secret, which will be created in the specified namespace. You will be prompted to enter the arbiter secret key-name and the target key in the Kubernetes secret. A patch file will be created in `<repo>/config/patches/kubernetes-secrets`.

Example secret configuration:
```yaml
# <repo>/config/patches/kubernetes-secrets/my-secret.yaml
# arbiter.x.y are just example names, you can call them anything, but they refer exactly to the secret keys in arbiter, they do not require an arbiter. prefix

namespace: cert-manager
name: cert-authority
keyMapping:
  tls.crt: arbiter.tls.cert
  tls.key: arbiter.tls.key
```

Another example:
```yaml
namespace: my_namespace
name: my_secret
keyMapping:
  username: users.me.name
  password: users.me.password
```

## Commands

### General Commands
- `init` - Initialize Talos project
- `bootstrap` - Bootstrap Talos on target system
- `post-configure` - Configure additional resources
- `apply` - Apply configuration changes
- `kubeconfig` - Manage Kubernetes configuration
- `info` - Display project information
- `export` - Export Talos configuration

### Secret Management
- `secrets set <key>` - Set/replace a secret
- `secrets remove <key>` - Remove a secret
- `secrets list` - List secrets
- `secrets apply <name>` - Apply a secret within Kubernetes deployment manifest

## Project Structure

The tool expects a specific project structure:
```
<project-directory>/
├── config/
│   ├── patches/
│   │   ├── machine/
│   │   └── kubernetes/
│   └── kubernetes-secrets/
├── .git/
└── config.yaml
```

## Security

All secret data is encrypted using the key material stored in `~/.arbiter/<project-name>/config.yaml`. It is crucial to backup this file securely.

## Contributing

Contributions are welcome! Please follow the project's development workflow:
1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.