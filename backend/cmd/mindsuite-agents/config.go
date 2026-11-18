package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"
)

// configName is the backend's config file. It holds settings only, never the Claude API key:
// apiKeyFile points to the key file encrypted for your Windows account (see backend/README.md).
const configName = "mindsuite.json"

type fileConfig struct {
	About     string   `json:"_about"`
	Addr      string   `json:"addr"`
	Runs      *string  `json:"runs"`
	CORS      []string `json:"cors"`
	AllowPath *bool    `json:"allowPath"`
	Models    struct {
		Strong   string `json:"strong"`
		Balanced string `json:"balanced"`
		Fast     string `json:"fast"`
	} `json:"models"`
	APIKeyFile string `json:"apiKeyFile"`
	path       string
}

var keyLike = regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{8,}`)

// loadConfig reads -config <file>, else mindsuite.json in the current folder, else next to the program.
// A missing default file is fine (built-in settings are used); a missing -config file is an error.
func loadConfig(args []string) fileConfig {
	explicit := ""
	for i, a := range args {
		if a == "-config" || a == "--config" {
			if i+1 < len(args) {
				explicit = args[i+1]
			}
		} else if v, ok := strings.CutPrefix(a, "-config="); ok {
			explicit = v
		} else if v, ok := strings.CutPrefix(a, "--config="); ok {
			explicit = v
		}
	}
	paths := []string{explicit}
	if explicit == "" {
		paths = []string{configName}
		if exe, err := os.Executable(); err == nil {
			paths = append(paths, filepath.Join(filepath.Dir(exe), configName))
		}
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			if explicit != "" {
				log.Fatalf("config file %s: %v", p, err)
			}
			continue
		}
		if keyLike.Match(b) {
			log.Fatalf("%s seems to contain a Claude API key. Remove it: the config file is committed with the code. Use apiKeyFile (an encrypted key file) or ANTHROPIC_API_KEY instead.", p)
		}
		var c fileConfig
		dec := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&c); err != nil {
			log.Fatalf("config file %s: %v", p, err)
		}
		c.path = p
		return c
	}
	return fileConfig{}
}

// apiKey returns the Claude API key and where it came from: ANTHROPIC_API_KEY first,
// else the encrypted key file named by apiKeyFile. The key itself is never logged.
func apiKey(c fileConfig) (string, string) {
	if k := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); k != "" {
		return k, "ANTHROPIC_API_KEY"
	}
	if c.APIKeyFile == "" {
		return "", ""
	}
	p := expandHome(c.APIKeyFile)
	k, err := readKeyFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("No Claude key: %s does not exist yet (see backend/README.md → Claude API key).", p)
		} else {
			log.Printf("No Claude key: could not read %s: %v", p, err)
		}
		return "", ""
	}
	return k, "encrypted key file " + p
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[1:])
		}
	}
	return p
}

var secureStringRe = regexp.MustCompile(`<SS[^>]*>\s*([0-9a-fA-F]+)\s*</SS>`)

// readKeyFile decrypts a key saved with PowerShell:
//
//	Read-Host "Paste Claude key" -AsSecureString | Export-Clixml "$HOME\.mindsuite\claude-key.xml"
func readKeyFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	m := secureStringRe.FindStringSubmatch(decodeText(b))
	if m == nil {
		return "", errors.New("it is not an encrypted key file made with Export-Clixml")
	}
	blob, err := hex.DecodeString(m[1])
	if err != nil {
		return "", err
	}
	plain, err := dpapiDecrypt(blob)
	if err != nil {
		return "", fmt.Errorf("Windows could not decrypt it (it only opens for the Windows user who saved it): %w", err)
	}
	u := make([]uint16, len(plain)/2)
	for i := range u {
		u[i] = uint16(plain[2*i]) | uint16(plain[2*i+1])<<8
	}
	key := strings.TrimSpace(string(utf16.Decode(u)))
	if key == "" {
		return "", errors.New("the saved key is empty")
	}
	return key, nil
}

// decodeText reads UTF-16 (what Windows PowerShell writes) or UTF-8.
func decodeText(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		b = b[2:]
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
		}
		return string(utf16.Decode(u))
	}
	return string(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")))
}
