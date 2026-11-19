package secrets

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/exadrift/go/statusline"
	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	pathArg    string
	projectArg string
)

// Command returns the 'secrets' command for managing secrets
// This command provides functionality for setting, removing, listing, and applying
// secrets within the arbiter system and Kubernetes deployments
func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "secrets",
		Short: "manage secrets",
		Long:  "manage secrets for Talos cluster configuration and Kubernetes deployments",
	}

	var setCmd = &cobra.Command{
		Use:   "set <key>",
		Short: "Set/replace a secret",
		Long:  "Set or replace a secret in the arbiter secret store with the given key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, cerr := os.Getwd()
			if cerr != nil {
				return cerr
			}
			var (
				err  error
				name = projectArg
			)
			if name == "" {
				name, err = config.GetNameRef(cwd)
				if err != nil {
					return err
				}
			}

			cfg, err := config.Load(name)
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true
			return setSecret(cfg, args[0], pathArg)
		}}

	setCmd.PersistentFlags().StringVar(&pathArg, "path", "", "optional path to secret file")

	var removeCmd = &cobra.Command{
		Use:   "remove <key>",
		Short: "Remove a secret",
		Long:  "Remove a secret from the arbiter secret store with the given key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, cerr := os.Getwd()
			if cerr != nil {
				return cerr
			}
			var (
				err  error
				name = projectArg
			)
			if name == "" {
				name, err = config.GetNameRef(cwd)
				if err != nil {
					return err
				}
			}

			cfg, err := config.Load(name)
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true
			return removeSecret(cfg, args[0])
		}}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "lists secrets",
		Long:  "List all secrets in the arbiter secret store with their creation timestamps",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, cerr := os.Getwd()
			if cerr != nil {
				return cerr
			}
			var (
				err  error
				name = projectArg
			)
			if name == "" {
				name, err = config.GetNameRef(cwd)
				if err != nil {
					return err
				}
			}

			cfg, err := config.Load(name)
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true
			return listSecrets(cfg)
		}}

	var applyCmd = &cobra.Command{
		Use:   "apply <name>",
		Short: "Apply a secret within the kubernetes deploy manifest",
		Long:  "Create a Kubernetes secret that references arbiter secrets for deployment in Kubernetes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, cerr := os.Getwd()
			if cerr != nil {
				return cerr
			}
			var (
				err  error
				name = projectArg
			)
			if name == "" {
				name, err = config.GetNameRef(cwd)
				if err != nil {
					return err
				}
			}

			cfg, err := config.Load(name)
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true
			namespace, _ := cmd.Flags().GetString("namespace")
			return applySecret(cfg, args[0], namespace)
		}}

	applyCmd.Flags().String("namespace", "", "namespace in which to install secret")
	_ = cobra.MarkFlagRequired(applyCmd.Flags(), "namespace")

	base.PersistentFlags().StringVar(
		&projectArg,
		"project",
		"",
		"Project name",
	)

	base.AddCommand(setCmd)
	base.AddCommand(removeCmd)
	base.AddCommand(listCmd)
	base.AddCommand(applyCmd)

	return base
}

// setSecret sets or replaces a secret in the configuration
// It reads secret data from a file path or stdin and stores it in the arbiter secret store
func setSecret(cfg *config.Config, key string, path string) error {
	var (
		secretBytes []byte
		err         error
	)
	if path != "" {
		secretBytes, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	} else {
		fmt.Print("Enter secret password: ")

		// term.ReadPassword reads input without echoing it back to the terminal
		secretBytes, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}

		// Print a newline since pressing Enter doesn't advance the cursor visually
		fmt.Println()
	}

	if err = cfg.Secrets().Set(key, secretBytes); err != nil {
		return err
	}

	if err = cfg.Secrets().Save(); err != nil {
		return err
	}

	fmt.Printf("✅ secret \"%s\" added\n", key)
	return nil
}

// removeSecret removes a secret from the configuration
// It deletes the secret from the arbiter secret store if it exists
func removeSecret(cfg *config.Config, key string) error {
	if !cfg.Secrets().Exists(key) {
		return nil
	}

	cfg.Secrets().Remove(key)

	if err := cfg.Secrets().Save(); err != nil {
		return err
	}

	fmt.Printf("✅ secret \"%s\" removed\n", key)
	return nil
}

// listSecrets lists all secrets in the configuration with their creation times
// It outputs a tabular format showing the key names and ages of secrets
func listSecrets(cfg *config.Config) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(w, "KEY\tAGE"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "---\t---"); err != nil {
		return err
	}
	for key, secret := range cfg.Secrets().Kv {
		dur := time.Duration.Round(time.Since(secret.CreatedAt), time.Second)
		if _, err := fmt.Fprintf(w, "%s\t%s\n", key, dur); err != nil {
			return err
		}
	}

	return w.Flush()
}

// applySecret creates a Kubernetes secret that references arbiter secrets
// It guides the user through mapping arbiter secret keys to Kubernetes secret keys
func applySecret(cfg *config.Config, name string, namespace string) error {
	sl := statusline.NewStatusLine()
	defer sl.Stop()

	kSecret := config.KubernetesSecretMapping{
		Name:       name,
		Namespace:  namespace,
		KeyMapping: map[string]string{},
	}

	for {
		var (
			keyName       string
			secretKeyName string
		)
		for {
			keyName = sl.Prompt(false, "what's the name of the key in the arbiter secret store?")
			keyName = strings.Trim(keyName, "\n \t")
			if !cfg.Secrets().Exists(keyName) {
				sl.Emit("❌ that key name doesn't exist")
			} else {
				break
			}
		}

		for {
			secretKeyName = sl.Prompt(false, "what's do you want the secret key to be named?")
			secretKeyName = strings.Trim(secretKeyName, "\n \t")
			if secretKeyName == "" {
				sl.Emit("❌ secret key name can't be empty")
			} else {
				break
			}
		}

		kSecret.KeyMapping[secretKeyName] = keyName

		if sl.Confirm(false, "would you like to add another k/v pair to the secret?") {
			continue
		} else {
			return cfg.WriteKubernetsSecret(&kSecret)
		}
	}
}
