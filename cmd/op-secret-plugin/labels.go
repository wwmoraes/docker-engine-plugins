package main

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

const (
	// LabelVault is the secret label key that holds the vault name
	LabelVault string = `connect.1password.io/vault`
	// LabelItem is the secret label key that holds the item ID
	LabelItem string = `connect.1password.io/item`
	// LabelField is the secret label key that holds the field name
	LabelField string = `connect.1password.io/field`
	// LabelReusable is the optional secret label key that sets a secret as single-use
	LabelReusable string = `connect.1password.io/reusable`
)

// labels contains all secret labels known and used by the driver. It
// includes both mandatory and optional keys
type labels struct {
	Vault    string `mapstructure:"connect.1password.io/vault"`
	Item     string `mapstructure:"connect.1password.io/item"`
	Field    string `mapstructure:"connect.1password.io/field"`
	Reusable *bool  `mapstructure:"connect.1password.io/reusable,omitempty"`
}

// newLabels unmarshal a labels map and validates if the mandatory keys are set
func newLabels(values map[string]string) (*labels, error) {
	var labels labels

	err := mapstructure.WeakDecode(values, &labels)

	if err == nil && labels.Vault == "" {
		err = fmt.Errorf("%s: %w", LabelVault, ErrLabelNotFound)
	}

	if err == nil && labels.Item == "" {
		err = fmt.Errorf("%s: %w", LabelItem, ErrLabelNotFound)
	}

	if err == nil && labels.Field == "" {
		err = fmt.Errorf("%s: %w", LabelField, ErrLabelNotFound)
	}

	return &labels, err
}
