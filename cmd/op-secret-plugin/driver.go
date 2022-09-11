package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/docker/go-plugins-helpers/secrets"
)

var (
	// ErrVaultNotFound represents a vault response that yields zero objects
	ErrVaultNotFound = errors.New("vault not found")
	// ErrAmbiguousVaultName represents a title vault response that yields multiple vaults
	ErrAmbiguousVaultName = errors.New("ambiguous Vault name")
	// ErrNilClient is returned when a new driver is created with a nil client
	ErrNilClient = errors.New("no client provided")
	// ErrLabelNotFound is returned when mandatory labels are not found
	ErrLabelNotFound = errors.New("label not found")
)

type onePasswordDriver struct {
	client connect.Client
}

// New wraps a 1Password Connect client as a Docker Engine secrets driver
func New(client connect.Client) (secrets.Driver, error) {
	if client == nil {
		return nil, ErrNilClient
	}

	return &onePasswordDriver{
		client: client,
	}, nil
}

func (driver *onePasswordDriver) getVaultByTitle(value string) (*onepassword.Vault, error) {
	vaults, err := driver.client.GetVaultsByTitle(value)
	if err != nil {
		return nil, err
	}

	switch len(vaults) {
	case 0:
		return nil, ErrVaultNotFound
	case 1:
		return &vaults[0], nil
	default:
		return nil, ErrAmbiguousVaultName
	}
}

// Get retrieves a secret value from 1Password
func (driver *onePasswordDriver) Get(req secrets.Request) secrets.Response {
	values, err := newLabels(req.SecretLabels)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("driver: %w", err))
		return secrets.Response{
			Err: err.Error(),
		}
	}

	vault, err := driver.getVaultByTitle(values.Vault)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("driver: %w", err))
		return secrets.Response{
			Err: err.Error(),
		}
	}

	item, err := driver.client.GetItemByTitle(values.Item, vault.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("driver: %w", err))
		return secrets.Response{
			Err: err.Error(),
		}
	}

	return secrets.Response{
		Value:      []byte(item.GetValue(values.Field)),
		DoNotReuse: values.Reusable != nil && !*values.Reusable,
	}
}
