package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var registerCmd = &cobra.Command{
	Use:   "register <name>",
	Short: "Register a new user and save the API key to the config file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		host := viper.GetString("host")
		if host == "" {
			fmt.Fprintln(os.Stderr, "error: 'host' not set — run 'chatatui init' first")
			os.Exit(1)
		}

		url := registerURL(host)

		body, _ := json.Marshal(map[string]string{"name": name})
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: request failed: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			var errBody map[string]string
			_ = json.NewDecoder(resp.Body).Decode(&errBody)
			fmt.Fprintf(os.Stderr, "error: server returned %d: %s\n", resp.StatusCode, errBody["error"])
			os.Exit(1)
		}

		var result struct {
			APIKey string `json:"api_key"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to parse response: %v\n", err)
			os.Exit(1)
		}

		if err := saveAPIKey(result.APIKey); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to save api_key to config: %v\n", err)
			fmt.Fprintf(os.Stderr, "your api_key: %s\n", result.APIKey)
			os.Exit(1)
		}

		fmt.Printf("registered as %q — api_key saved to config\n", name)
	},
}

func registerURL(host string) string {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	return host + "/register"
}

func saveAPIKey(apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".chatatui.toml")

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "api_key") {
			lines[i] = fmt.Sprintf("api_key = %q", apiKey)
			replaced = true
			break
		}
	}
	if !replaced {
		return fmt.Errorf("api_key field not found in %s", path)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
}

func init() {
	rootCmd.AddCommand(registerCmd)
}
