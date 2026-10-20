package argocd

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

//go:embed namespace.yaml
var namespaceYaml []byte

const (
	versionURL  = "https://raw.githubusercontent.com/argoproj/argo-cd/stable/VERSION"
	manifestURL = "https://raw.githubusercontent.com/argoproj/argo-cd/v%s/manifests/install.yaml"
)

func fetchURL(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}

	return body, nil
}

func getInstallManifest() ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	versionData, err := fetchURL(client, versionURL)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch argo cd version: %w", err)
	}

	version := strings.TrimSpace(string(versionData))
	if version == "" {
		return nil, fmt.Errorf("argo version reported empty string")
	}

	url := fmt.Sprintf(manifestURL, version)
	manifest, err := fetchURL(client, url)
	if err != nil {
		return nil, fmt.Errorf("error fetching argocd install manifest: %w", err)
	}

	return manifest, nil
}

func GetArgoCdFullManifest() ([]byte, error) {
	installMf, err := getInstallManifest()
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf("%s\n---\n%s\n", string(installMf), string(namespaceYaml))), nil
}
