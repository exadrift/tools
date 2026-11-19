# Arbiter Architecture Documentation

This document describes the architecture and design of the arbiter tool for Talos cluster management.

## Overview

Arbiter is a command-line tool that provides a guided workflow for managing single-node Talos clusters with GitOps principles. It focuses on:

- Secure secret management
- Automated ArgoCD deployment
- Git-friendly cluster configuration
- Reproducible cluster setup

## Project Structure

```
arbiter/
├── main.go                 # Entry point and command setup
├── README.md              # Main documentation
├── CHANGELOG.md           # Version history
├── go.mod                 # Go module dependencies
├── internal/              # Internal packages
│   ├── cmd/               # Command implementations
│   │   ├── initc/         # Project initialization
│   │   ├── bootstrap/     # Cluster bootstrap
│   │   ├── post-configure/ # Post-configuration
│   │   ├── apply/         # Apply changes
│   │   ├── secrets/       # Secret management
│   │   ├── kubeconfig/    # Kubernetes config management
│   │   ├── info/          # Information display
│   │   ├── export/        # Export functionality
│   │   ├── list/          # List resources
│   │   └── remove/        # Remove resources
│   ├── config/            # Configuration handling
│   ├── crypto/            # Encryption/decryption utilities
│   ├── git/               # Git operations
│   ├── kube/              # Kubernetes operations
│   ├── talossdk/          # Talos SDK integration
│   └── utils/             # Utility functions
└── templates/             # Template files for configuration
```

## Core Components

### 1. Command Structure

Arbiter uses Cobra for CLI command management with a hierarchical structure:

- `init` - Initializes a new project
- `bootstrap` - Boots up the Talos cluster
- `post-configure` - Configures additional resources
- `apply` - Applies configuration changes
- `secrets` - Manages secret storage and Kubernetes integration
- `kubeconfig` - Manages Kubernetes configuration
- `info` - Displays project information
- `export` - Exports Talos configuration
- `list` - Lists resources
- `remove` - Removes resources

### 2. Configuration Management

The configuration system handles:
- Project-level settings stored in `~/.arbiter/<project-name>/config.yaml`
- Secret management with encryption/decryption
- Machine configuration templates
- Kubernetes secret mappings

### 3. Secret Management System

Arbiter implements a two-layer secret management:
1. **Arbiter Secrets**: Encrypted secrets stored in the configuration directory
2. **Kubernetes Secrets**: Mappings that create Kubernetes secrets referencing arbiter secrets

### 4. Integration Points

- **Talos SDK**: Direct communication with Talos clusters
- **ArgoCD**: GitOps deployment through Helm chart installation
- **Helm**: Package management for ArgoCD installation
- **Git**: Version control for cluster configuration
- **Kubernetes**: Configuration and deployment management

## Workflow

### Initialization Process
1. User creates a Git repository for cluster configuration
2. `arbiter init` creates project structure and key material
3. User provides cluster IP address and project name
4. Configuration is stored securely in `~/.arbiter/<project-name>`

### Bootstrap Process
1. `arbiter bootstrap` connects to Talos maintenance mode
2. Validates cluster state and discovers available disks
3. Applies machine configuration to the target system
4. Waits for cluster to become healthy
5. Installs ArgoCD via Helm
6. Sets up GitOps application manifests

### Secret Management Workflow
1. `arbiter secrets set <key>` stores a secret
2. `arbiter secrets apply <name>` creates Kubernetes secret mapping
3. `arbiter apply` deploys secrets as Kubernetes manifests

## Security Considerations

- All secret data is encrypted using the project's key material
- Key material is stored in `config.yaml` which must be backed up securely
- Secrets are never stored in plaintext in Git repositories
- The encryption key is derived from the project configuration

## Design Principles

1. **GitOps First**: All configuration is stored in Git for version control
2. **Security by Design**: Secrets are encrypted and never exposed in plaintext
3. **Guided Workflow**: Interactive prompts help users through complex processes
4. **Reproducibility**: Configuration can be reproduced on any system
5. **Minimal Dependencies**: Uses standard tools where possible