package main

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/docker/go-plugins-helpers/secrets"
	"github.com/mitchellh/mapstructure"
)

func TestNew(t *testing.T) {
	emptyClient := connect.NewClient("http://localhost", "")

	type args struct {
		client connect.Client
	}
	tests := []struct {
		name    string
		args    args
		want    secrets.Driver
		wantErr bool
	}{
		{
			name: "valid client",
			args: args{
				client: emptyClient,
			},
			want: &onePasswordDriver{
				client: emptyClient,
			},
			wantErr: false,
		},
		{
			name: "nil client",
			args: args{
				client: nil,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnePasswordDriver_Get(t *testing.T) {
	backend := newBackend(t, mockToken)
	defer backend.Close()
	client := newClient(t, backend)
	driver := newDriver(t, client)

	reusableValueInvalid := "test"

	type args struct {
		req secrets.Request
	}
	tests := []struct {
		name string
		args args
		want secrets.Response
	}{
		{
			name: "valid request",
			args: args{
				req: secrets.Request{
					SecretLabels: map[string]string{
						LabelVault: mockVaultTitle,
						LabelItem:  mockItemTitle,
						LabelField: mockItemFieldLabel,
					},
				},
			},
			want: secrets.Response{
				DoNotReuse: false,
				Err:        "",
				Value:      []byte(mockItemFieldValue),
			},
		},
		{
			name: "missing vault label",
			args: args{
				req: secrets.Request{
					SecretLabels: map[string]string{
						// LabelVault: mockVaultTitle,
						LabelItem:  mockItemTitle,
						LabelField: mockItemFieldLabel,
					},
				},
			},
			want: secrets.Response{
				DoNotReuse: false,
				Err:        fmt.Errorf("%s: %w", LabelVault, ErrLabelNotFound).Error(),
				Value:      nil,
			},
		},
		{
			name: "missing item label",
			args: args{
				req: secrets.Request{
					SecretLabels: map[string]string{
						LabelVault: mockVaultTitle,
						// LabelItem:  mockItemTitle,
						LabelField: mockItemFieldLabel,
					},
				},
			},
			want: secrets.Response{
				DoNotReuse: false,
				Err:        fmt.Errorf("%s: %w", LabelItem, ErrLabelNotFound).Error(),
				Value:      nil,
			},
		},
		{
			name: "missing field label",
			args: args{
				req: secrets.Request{
					SecretLabels: map[string]string{
						LabelVault: mockVaultTitle,
						LabelItem:  mockItemTitle,
						// LabelField: mockItemFieldLabel,
					},
				},
			},
			want: secrets.Response{
				DoNotReuse: false,
				Err:        fmt.Errorf("%s: %w", LabelField, ErrLabelNotFound).Error(),
				Value:      nil,
			},
		},
		{
			name: "missing item",
			args: args{
				req: secrets.Request{
					SecretLabels: map[string]string{
						LabelVault: mockVaultTitle,
						LabelItem:  mockItemTitleNonExistent,
						LabelField: mockItemFieldLabel,
					},
				},
			},
			want: secrets.Response{
				DoNotReuse: false,
				Err:        fmt.Errorf("Found %d item(s) in vault %q with title %q", 0, mockVaultUUID, mockItemTitleNonExistent).Error(),
				Value:      nil,
			},
		},
		{
			name: "incompatible label value",
			args: args{
				req: secrets.Request{
					SecretLabels: map[string]string{
						LabelVault:    mockVaultTitle,
						LabelItem:     mockItemTitleNonExistent,
						LabelField:    mockItemFieldLabel,
						LabelReusable: reusableValueInvalid,
					},
				},
			},
			want: secrets.Response{
				DoNotReuse: false,
				Err: (&mapstructure.Error{
					Errors: []string{
						fmt.Sprintf(`cannot parse '%s' as bool: %s`, LabelReusable, (&strconv.NumError{
							Func: "ParseBool",
							Num:  reusableValueInvalid,
							Err:  strconv.ErrSyntax,
						}).Error()),
					},
				}).Error(),
				Value: nil,
			},
		},
		{
			name: "dupe vault",
			args: args{
				req: secrets.Request{
					SecretLabels: map[string]string{
						LabelVault: mockVaultDupeTitle,
						LabelItem:  mockItemTitle,
						LabelField: mockItemFieldLabel,
					},
				},
			},
			want: secrets.Response{
				DoNotReuse: false,
				Err:        ErrAmbiguousVaultName.Error(),
				Value:      nil,
			},
		},
		{
			name: "non-existent vault",
			args: args{
				req: secrets.Request{
					SecretLabels: map[string]string{
						LabelVault: "non-existent",
						LabelItem:  mockItemTitle,
						LabelField: mockItemFieldLabel,
					},
				},
			},
			want: secrets.Response{
				DoNotReuse: false,
				Value:      nil,
				Err:        ErrVaultNotFound.Error(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := driver.Get(tt.args.req); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnePasswordDriver.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_onePasswordDriver_getVaultByTitle(t *testing.T) {
	backend := newBackend(t, mockToken)
	defer backend.Close()
	client := newClient(t, backend)
	driver := &onePasswordDriver{
		client: client,
	}

	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		want    *onepassword.Vault
		wantErr bool
	}{
		// {
		// 	name: "valid title",
		// 	args: args{
		// 		value: mockVaultTitle,
		// 	},
		// 	want:    &backend.vaults[0],
		// 	wantErr: false,
		// },
		{
			name: "invalid title",
			args: args{
				value: "",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := driver.getVaultByTitle(tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("onePasswordDriver.getVaultByTitle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("onePasswordDriver.getVaultByTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}
