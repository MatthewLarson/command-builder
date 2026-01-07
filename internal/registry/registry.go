package registry

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"command-builder/internal/definitions"
)

const registryBaseURL = "https://raw.githubusercontent.com/MatthewLarson/command-builder-definitions/main/definitions/"

type Client struct {
	httpClient *http.Client
	defManager *definitions.Manager
}

func NewClient(defMgr *definitions.Manager) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		defManager: defMgr,
	}
}

// FetchDefinition attempts to download a definition file from the registry.
// If found, it saves it to the local definition directory.
func (c *Client) FetchDefinition(commandName string) error {
	url := registryBaseURL + commandName + ".yaml"
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to contact registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("definition not found in registry")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned status: %d", resp.StatusCode)
	}

	// We can't access defDir directly from manager if it's private, strictly speaking.
	// But assuming we know the path or added a getter.
	// For simplicity, let's just re-derive the path as Definition Manager does.
	// In a real app we'd expose GetDefDir().
	home, _ := os.UserHomeDir()
	defPath := filepath.Join(home, ".config", "command-builder", "definitions", commandName+".yaml")

	out, err := os.Create(defPath)
	if err != nil {
		return fmt.Errorf("failed to create local definition file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write definition file: %w", err)
	}

	return nil
}
