package main

import (
	"fmt"
	"net/url"
	"os"
)

const (
	// EnvHost is the host URL environment variable name
	EnvHost string = `OP_CONNECT_HOST`
	// EnvToken is the token environment variable name
	EnvToken string = `OP_CONNECT_TOKEN`
	// EnvTokenFile is the token file environment variable name
	EnvTokenFile string = `OP_CONNECT_TOKEN_FILE`
)

// config contains all settings used by the main application
type config struct {
	URL   string
	Token string
}

// newConfig loads settings from the environment
func newConfig() (*config, error) {
	host, err := getHost()
	if err != nil {
		return nil, err
	}

	token, err := getToken()
	if err != nil {
		return nil, err
	}

	return &config{
		URL:   host,
		Token: token,
	}, nil
}

func getHost() (string, error) {
	host, exists := os.LookupEnv(EnvHost)
	if !exists {
		return "", fmt.Errorf("%s not set", EnvHost)
	}

	if host == "" {
		return "", fmt.Errorf("%s must not be empty", EnvHost)
	}

	_, err := url.ParseRequestURI(host)
	if err != nil {
		return "", fmt.Errorf("failed to parse connect host: %w", err)
	}

	return host, nil
}

func getToken() (string, error) {
	token, exists := os.LookupEnv(EnvToken)
	if !exists || token == "" {
		var err error
		token, err = getTokenFromFile()
		if err != nil {
			return "", err
		}
	}

	if token == "" {
		return "", fmt.Errorf("token must not be empty")
	}

	return token, nil
}

func getTokenFromFile() (string, error) {
	tokenFile, exists := os.LookupEnv(EnvTokenFile)
	if !exists {
		return "", fmt.Errorf("%s not set", EnvTokenFile)
	}
	if tokenFile == "" {
		return "", fmt.Errorf("%s must not be empty", EnvTokenFile)
	}

	return readFileString(tokenFile)
}
