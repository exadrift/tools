package git

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func GetRemoteOriginURL(configPath string) (string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return "", fmt.Errorf("could not open git config: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	inRemoteOriginBlock := false
	var rawURL string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == `[remote "origin"]` {
			inRemoteOriginBlock = true
			continue
		}

		if inRemoteOriginBlock && strings.HasPrefix(line, "[") {
			break
		}

		if inRemoteOriginBlock && strings.HasPrefix(line, "url =") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				rawURL = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if rawURL == "" {
		return "", fmt.Errorf("remote origin URL not found in config")
	}

	return convertToHTTPS(rawURL), nil
}

// convertToHTTPS transforms an SSH git URL or an HTTP URL into a strict https:// URL
func convertToHTTPS(url string) string {
	// Already HTTPS or HTTP
	if strings.HasPrefix(url, "https://") {
		return url
	}
	if strings.HasPrefix(url, "http://") {
		return "https://" + strings.TrimPrefix(url, "http://")
	}

	// Handle SSH format: git@github.com:user/repo.git or ssh://git@://github.com
	cleaned := url
	cleaned = strings.TrimPrefix(cleaned, "ssh://")

	// Remove the 'git@' or 'user@' prefix if present
	if idx := strings.Index(cleaned, "@"); idx != -1 {
		cleaned = cleaned[idx+1:]
	}

	// Git SSH syntax uses a colon separating host and path (e.g., github.com:user/repo)
	// We replace the first colon after the domain name with a slash '/'
	cleaned = strings.Replace(cleaned, ":", "/", 1)

	// Clean up trailing .git suffix if you want a clean web URL (optional)
	cleaned = strings.TrimSuffix(cleaned, ".git")

	return "https://" + cleaned
}
