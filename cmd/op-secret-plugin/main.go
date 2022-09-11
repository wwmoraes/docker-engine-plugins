package main

import (
	"fmt"
	"os"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/docker/go-plugins-helpers/secrets"
	"github.com/wwmoraes/docker-engine-plugins/internal/common"
)

// PluginName represents the name for the driver's socket
const PluginName string = `op`

// readFileString reads an entire file and returns the bytes as string
func readFileString(name string) (string, error) {
	info, err := os.Stat(name)
	if err != nil {
		return "", err
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a file", name)
	}

	fileBytes, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}

	return string(fileBytes), nil
}

func main() {
	config, err := newConfig()
	common.Assert(err)

	client := connect.NewClient(config.URL, config.Token)

	driver, err := New(client)
	common.Assert(err)

	handler := secrets.NewHandler(driver)
	common.Assert(handler.ServeUnix(PluginName, 0))
}
