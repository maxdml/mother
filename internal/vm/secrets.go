package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const secretsPath = ".mother/secrets.yaml"

// DecryptSecrets decrypts ~/.mother/secrets.yaml with SOPS and writes
// the global + service-specific secrets as KEY=VALUE lines to a temp file.
// The caller is responsible for cleaning up the returned file.
// Returns empty string (no error) if the secrets file does not exist.
func DecryptSecrets(service string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}

	sopsFile := filepath.Join(homeDir, secretsPath)
	if _, err := os.Stat(sopsFile); os.IsNotExist(err) {
		return "", nil
	}

	cmd := exec.Command("sops", "-d", "--output-type", "json", sopsFile)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("sops decrypt: %w", err)
	}

	var data map[string]map[string]string
	if err := json.Unmarshal(out, &data); err != nil {
		return "", fmt.Errorf("parsing decrypted secrets: %w", err)
	}

	// Merge global + service-specific secrets
	merged := make(map[string]string)
	for k, v := range data["global"] {
		merged[k] = v
	}
	for k, v := range data[service] {
		merged[k] = v
	}

	if len(merged) == 0 {
		return "", nil
	}

	// Write as KEY=VALUE env file
	tmpFile, err := os.CreateTemp("", "mother-secrets-*.env")
	if err != nil {
		return "", fmt.Errorf("creating secrets temp file: %w", err)
	}

	for k, v := range merged {
		fmt.Fprintf(tmpFile, "%s=%s\n", k, v)
	}
	tmpFile.Close()

	os.Chmod(tmpFile.Name(), 0600)

	return tmpFile.Name(), nil
}
