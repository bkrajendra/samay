// Package config loads server configuration from environment variables,
// optionally seeded from a .env file. Real environment variables always
// take precedence over .env file contents.
package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Username     string
	Password     string
	ChronySocket string
	ListenAddr   string
	CookieSecure bool
}

// Load reads .env (if present) into the process environment, then builds a
// Config from the environment.
func Load(envPath string) Config {
	loadDotEnv(envPath)

	return Config{
		Username:     getEnv("CONSOLE_USERNAME", "admin"),
		Password:     getEnv("CONSOLE_PASSWORD", ""),
		ChronySocket: getEnv("CHRONY_SOCKET", "/run/chrony/chronyd.sock"),
		ListenAddr:   getEnv("LISTEN_ADDR", ":8080"),
		CookieSecure: getEnv("COOKIE_SECURE", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// loadDotEnv parses simple KEY=VALUE lines from path into the process
// environment, skipping blank lines and lines starting with '#'. It never
// overrides a variable that's already set in the real environment. Missing
// files are silently ignored.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, alreadySet := os.LookupEnv(key); !alreadySet {
			os.Setenv(key, value)
		}
	}
}
