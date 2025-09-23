// cmd/secrets/main.go
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	secretsDir     = "secrets"
	recipientsFile = ".age-recipients"
	defaultKeyPath = "~/.config/age/keys.txt"
)

var rootCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage age-encrypted secrets",
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Initialize secrets directory and recipients file",
	Run: func(cmd *cobra.Command, args []string) {
		if err := os.MkdirAll(secretsDir, 0755); err != nil {
			fmt.Printf("Error creating directory: %v\n", err)
			os.Exit(1)
		}

		if _, err := os.Stat(recipientsFile); os.IsNotExist(err) {
			content := "# Add age public keys, one per line\nage1k0sc4ugaxzpav2rs8cmugwthaa3tpuzygvax8u84m6sm9ldh737qspv058\n"
			if err := ioutil.WriteFile(recipientsFile, []byte(content), 0644); err != nil {
				fmt.Printf("Error creating recipients file: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Created recipients file")
		}
		fmt.Println("✓ Secrets directory ready")
	},
}

var addCmd = &cobra.Command{
	Use:   "add [secret-name]",
	Short: "Add a new secret",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		secretName := args[0]
		if !strings.HasSuffix(secretName, ".age") {
			secretName += ".age"
		}

		fmt.Print("Enter secret value: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		value := scanner.Text()

		secretPath := filepath.Join(secretsDir, secretName)
		if err := encryptSecret(value, secretPath); err != nil {
			fmt.Printf("Error encrypting secret: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Secret '%s' encrypted\n", secretName)
	},
}

var editCmd = &cobra.Command{
	Use:   "edit [secret-name]",
	Short: "Edit an existing secret",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getSecretNames(), cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		secretName := args[0]
		if !strings.HasSuffix(secretName, ".age") {
			secretName += ".age"
		}

		secretPath := filepath.Join(secretsDir, secretName)

		// Create temp file
		tempFile, err := ioutil.TempFile("", "secret-*.txt")
		if err != nil {
			fmt.Printf("Error creating temp file: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(tempFile.Name())

		// Decrypt existing content if file exists
		if _, err := os.Stat(secretPath); err == nil {
			content, err := decryptSecret(secretPath)
			if err != nil {
				fmt.Printf("Error decrypting secret: %v\n", err)
				os.Exit(1)
			}
			tempFile.WriteString(content)
		}
		tempFile.Close()

		// Open editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}

		editCmd := exec.Command(editor, tempFile.Name())
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr

		if err := editCmd.Run(); err != nil {
			fmt.Printf("Error running editor: %v\n", err)
			os.Exit(1)
		}

		// Read edited content
		content, err := ioutil.ReadFile(tempFile.Name())
		if err != nil {
			fmt.Printf("Error reading temp file: %v\n", err)
			os.Exit(1)
		}

		// Encrypt and save
		if err := encryptSecret(string(content), secretPath); err != nil {
			fmt.Printf("Error encrypting secret: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Secret '%s' updated\n", secretName)
	},
}

var getCmd = &cobra.Command{
	Use:   "get [secret-name]",
	Short: "Get a secret value",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getSecretNames(), cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		secretName := args[0]
		if !strings.HasSuffix(secretName, ".age") {
			secretName += ".age"
		}

		secretPath := filepath.Join(secretsDir, secretName)
		content, err := decryptSecret(secretPath)
		if err != nil {
			fmt.Printf("Error decrypting secret: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(content)
	},
}

func encryptSecret(value, path string) error {
	cmd := exec.Command("age", "-R", recipientsFile, "-o", path)
	cmd.Stdin = strings.NewReader(value)
	return cmd.Run()
}

func decryptSecret(path string) (string, error) {
	keyPath := strings.Replace(defaultKeyPath, "~", os.Getenv("HOME"), 1)
	cmd := exec.Command("age", "-d", "-i", keyPath, path)
	output, err := cmd.Output()
	return string(output), err
}

func getSecretNames() []string {
	files, err := filepath.Glob(filepath.Join(secretsDir, "*.age"))
	if err != nil {
		return nil
	}

	var names []string
	for _, file := range files {
		name := filepath.Base(file)
		names = append(names, name)
	}
	return names
}

func main() {
	rootCmd.AddCommand(generateCmd, addCmd, editCmd, getCmd)

	// Add completion command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				return
			}
			switch args[0] {
			case "bash":
				rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
