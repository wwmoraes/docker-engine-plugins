package main

import (
	"fmt"
	"os"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/docker/go-plugins-helpers/secrets"
	"github.com/mitchellh/mapstructure"
)

const (
	LabelVault string = "connect.1password.io/vault"
	LabelItem  string = "connect.1password.io/item"
	LabelField string = "connect.1password.io/field"
	// LabelReusable string = "connect.1password.io/reusable"
)

type OnePasswordDriver struct {
	client connect.Client
}

type LabelValues struct {
	Vault    string `mapstructure:"connect.1password.io/vault"`
	Item     string `mapstructure:"connect.1password.io/item"`
	Field    string `mapstructure:"connect.1password.io/field"`
	Reusable *bool  `mapstructure:"connect.1password.io/reusable,omitempty"`
}

func (driver *OnePasswordDriver) getVaultByTitle(value string) (*onepassword.Vault, error) {
	vaults, err := driver.client.GetVaultsByTitle(value)
	if err != nil {
		return nil, err
	}

	if len(vaults) != 1 {
		return nil, fmt.Errorf("ambiguous vault name: %s", value)
	}

	return &vaults[0], nil
}

func (driver *OnePasswordDriver) getLabelValues(values map[string]string) (*LabelValues, error) {
	var labelValues LabelValues
	err := mapstructure.WeakDecode(values, &labelValues)
	if err != nil {
		return nil, err
	}

	if labelValues.Vault == "" {
		return nil, fmt.Errorf("%s label not set", LabelVault)
	}

	if labelValues.Item == "" {
		return nil, fmt.Errorf("%s label not set", LabelItem)
	}

	if labelValues.Field == "" {
		return nil, fmt.Errorf("%s label not set", LabelField)
	}

	return &labelValues, nil
}

func (driver *OnePasswordDriver) Get(req secrets.Request) secrets.Response {
	values, err := driver.getLabelValues(req.SecretLabels)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return secrets.Response{
			Err: err.Error(),
		}
	}

	vault, err := driver.getVaultByTitle(values.Vault)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return secrets.Response{
			Err: err.Error(),
		}
	}

	item, err := driver.client.GetItemByTitle(values.Item, vault.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return secrets.Response{
			Err: err.Error(),
		}
	}

	return secrets.Response{
		Value:      []byte(item.GetValue(values.Field)),
		DoNotReuse: values.Reusable != nil && !*values.Reusable,
	}
}

func main() {
	opConnectURL, exists := os.LookupEnv("OP_CONNECT_HOST")
	if !exists {
		fmt.Fprintln(os.Stderr, fmt.Errorf("OP_CONNECT_HOST not set"))
		os.Exit(1)
	}

	opConnectTokenFile := os.Getenv("OP_CONNECT_TOKEN_FILE")
	if opConnectTokenFile == "" {
		fmt.Fprintln(os.Stderr, fmt.Errorf("OP_CONNECT_TOKEN_FILE not set"))
		os.Exit(1)
	}

	info, err := os.Stat(opConnectTokenFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if !info.Mode().IsRegular() {
		fmt.Fprintln(os.Stderr, fmt.Errorf("%s is not a file", opConnectTokenFile))
		os.Exit(1)
	}

	opConnectTokenBytes, err := os.ReadFile(opConnectTokenFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := connect.NewClient(opConnectURL, string(opConnectTokenBytes))

	driver := OnePasswordDriver{
		client: client,
	}

	handler := secrets.NewHandler(&driver)
	err = handler.ServeUnix("op", 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
