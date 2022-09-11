package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/docker/go-plugins-helpers/secrets"
)

const (
	mockHost                 string = `http://unix`
	mockToken                string = `header.payload.signature`
	mockVaultUUID            string = `ca1fnkquspvpskw53qlyuflt7v`
	mockVaultTitle           string = `Test`
	mockVaultDupe1UUID       string = `4lpj4x1z0jmbkijsfjgkqowxlj`
	mockVaultDupe2UUID       string = `28ik03bjw1o0zqfmnkhjo4c3hi`
	mockVaultDupeTitle       string = `Dupe`
	mockItemUUID             string = `h6fuuu51rq1e34gibd7cow2jzp`
	mockItemTitle            string = `Item`
	mockItemFieldLabel       string = `Field`
	mockItemFieldValue       string = `dolor sit amet`
	mockItemTitleNonExistent string = `Non-existent`
)

var (
	opErrInvalidVaultUUID = onepassword.Error{
		StatusCode: http.StatusBadRequest,
		Message:    "Invalid Vault UUID",
	}
	opErrInvalidItemUUID = onepassword.Error{
		StatusCode: http.StatusBadRequest,
		Message:    "Invalid Item UUID",
	}
	// TODO test request without token
	opErrInvalidBearerToken = onepassword.Error{
		StatusCode: http.StatusUnauthorized,
		Message:    "Invalid bearer token",
	}
	opErrSomethingWentWrong = onepassword.Error{
		StatusCode: http.StatusInternalServerError,
		Message:    "Something went wrong",
	}
)

type opBackend struct {
	tb         testing.TB
	client     *http.Client
	server     *http.Server
	listener   net.Listener
	titleRegex *regexp.Regexp
	authRegex  *regexp.Regexp

	vaults []onepassword.Vault
	items  map[string][]onepassword.Item

	token string
}

func newBackend(tb testing.TB, token string) *opBackend {
	tb.Helper()

	// generate temporary file name to be used as the unix socket
	fd, err := os.CreateTemp("", "op-secret-plugin-*")
	if err != nil {
		tb.Fatal(err)
	}
	fd.Close()

	err = os.Remove(fd.Name())
	if err != nil {
		tb.Fatal(err)
	}

	// generate unix socket address
	addr, err := net.ResolveUnixAddr("unix", fd.Name())
	if err != nil {
		tb.Fatal(err)
	}

	// create unix socket listener
	listener, err := net.ListenUnix(addr.Network(), addr)
	if err != nil {
		tb.Fatal(err)
	}

	// create socket client
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.DialUnix("unix", nil, addr)
			},
		},
	}

	backend := opBackend{
		tb:       tb,
		client:   &client,
		listener: listener,
		items: map[string][]onepassword.Item{
			mockVaultUUID: {
				{
					ID:    mockItemUUID,
					Title: mockItemTitle,
					Fields: []*onepassword.ItemField{
						{
							Label: mockItemFieldLabel,
							Value: mockItemFieldValue,
						},
					},
					Vault: onepassword.ItemVault{
						ID: mockVaultUUID,
					},
				},
			},
		},
		vaults: []onepassword.Vault{
			{
				AttrVersion:    1,
				ContentVersoin: 1,
				CreatedAt:      time.Now().Round(0),
				Description:    "lorem ipsum",
				ID:             mockVaultUUID,
				Items:          1,
				Name:           mockVaultTitle,
				Type:           onepassword.PersonalVault,
				UpdatedAt:      time.Now().Round(0),
			},
			{
				AttrVersion:    1,
				ContentVersoin: 1,
				CreatedAt:      time.Now().Round(0),
				Description:    "lorem ipsum",
				ID:             mockVaultDupe1UUID,
				Items:          1,
				Name:           mockVaultDupeTitle,
				Type:           onepassword.PersonalVault,
				UpdatedAt:      time.Now().Round(0),
			},
			{
				AttrVersion:    1,
				ContentVersoin: 1,
				CreatedAt:      time.Now().Round(0),
				Description:    "lorem ipsum",
				ID:             mockVaultDupe2UUID,
				Items:          1,
				Name:           mockVaultDupeTitle,
				Type:           onepassword.PersonalVault,
				UpdatedAt:      time.Now().Round(0),
			},
		},
		token:      token,
		titleRegex: regexp.MustCompile(`title eq "([^"]+)"`),
		authRegex:  regexp.MustCompile(`Bearer (.*)`),
	}

	go backend.Serve(tb) //nolint:errcheck

	return &backend
}

func newClient(tb testing.TB, backend *opBackend) connect.Client {
	config, err := newConfig()
	if err != nil {
		tb.Fatal(err)
	}

	// the connect SDK constructor doesn't allow passing a custom client or a
	// constructor functor, but it does use the default client, so...
	http.DefaultClient = backend.client
	client := connect.NewClient(config.URL, config.Token)
	http.DefaultClient = &http.Client{}

	return client
}

func newDriver(tb testing.TB, client connect.Client) secrets.Driver {
	tb.Helper()

	driver, err := New(client)
	if err != nil {
		tb.Fatal(err)
	}

	return driver
}

func (backend *opBackend) writeApiError(w http.ResponseWriter, opErr *onepassword.Error) {
	backend.tb.Helper()

	data, err := json.Marshal(opErr)
	if err != nil {
		backend.tb.Fatal(err)
	}

	w.WriteHeader(opErr.StatusCode)
	w.Write(data) //nolint:errcheck
}

func (backend *opBackend) writeData(w http.ResponseWriter, data interface{}) {
	backend.tb.Helper()

	dataBytes, err := json.Marshal(data)
	if err != nil {
		backend.tb.Fatal(err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(dataBytes) //nolint:errcheck
}

func (backend *opBackend) assertAuthorization(req *http.Request) *onepassword.Error {
	backend.tb.Helper()

	authValue := req.Header.Get("Authorization")
	if authValue == "" {
		return &opErrInvalidBearerToken
	}

	matches := backend.authRegex.FindSubmatch([]byte(authValue))
	if matches == nil {
		return &opErrInvalidBearerToken
	}

	token := strings.TrimSpace(string(matches[1]))
	if token != backend.token {
		return &opErrInvalidBearerToken
	}

	return nil
}

func (backend *opBackend) ItemHandler(vaultUUID string, itemUUID string) http.HandlerFunc {
	backend.tb.Helper()

	return func(w http.ResponseWriter, req *http.Request) {
		backend.tb.Helper()

		if err := backend.assertAuthorization(req); err != nil {
			backend.writeApiError(w, err)
			return
		}

		items, exists := backend.items[vaultUUID]
		if !exists {
			backend.writeApiError(w, &opErrInvalidVaultUUID)
			return
		}

		for _, item := range items {
			if item.ID == itemUUID {
				backend.writeData(w, item)
				return
			}
		}

		backend.writeApiError(w, &opErrInvalidItemUUID)
	}
}

func (backend *opBackend) ItemsHandler(vaultUUID string) http.HandlerFunc {
	backend.tb.Helper()

	return func(w http.ResponseWriter, req *http.Request) {
		backend.tb.Helper()

		if err := backend.assertAuthorization(req); err != nil {
			backend.writeApiError(w, err)
			return
		}

		filter := req.URL.Query().Get("filter")
		items, exists := backend.items[vaultUUID]
		if !exists {
			backend.writeApiError(w, &opErrInvalidVaultUUID)
			return
		}

		if filter == "" {
			backend.writeData(w, items)
			return
		}

		match := backend.titleRegex.FindSubmatch([]byte(filter))
		if match == nil {
			backend.writeData(w, []interface{}{})
			return
		}

		name := strings.TrimSpace(string(match[1]))
		resultItems := make([]onepassword.Item, 0, len(items))
		for _, item := range items {
			if item.Title != name {
				continue
			}

			resultItems = append(resultItems, item)
		}

		backend.writeData(w, resultItems)
	}
}

func (backend *opBackend) VaultsHandler() http.HandlerFunc {
	backend.tb.Helper()

	return func(w http.ResponseWriter, req *http.Request) {
		backend.tb.Helper()

		if err := backend.assertAuthorization(req); err != nil {
			backend.writeApiError(w, err)
			return
		}

		filter := req.URL.Query().Get("filter")
		if filter == "" {
			backend.writeData(w, backend.vaults)
			return
		}

		match := backend.titleRegex.FindSubmatch([]byte(filter))
		if match == nil {
			backend.writeApiError(w, &opErrSomethingWentWrong)
			return
		}

		name := strings.TrimSpace(string(match[1]))
		if name == "" {
			backend.writeApiError(w, &opErrSomethingWentWrong)
			return
		}

		resultVaults := make([]onepassword.Vault, 0, len(backend.vaults))
		for _, vault := range backend.vaults {
			if vault.Name != name {
				continue
			}

			resultVaults = append(resultVaults, vault)
		}

		backend.writeData(w, resultVaults)
	}
}

func (backend *opBackend) FallbackHandler() http.HandlerFunc {
	backend.tb.Helper()

	return func(w http.ResponseWriter, req *http.Request) {
		backend.tb.Helper()

		backend.tb.Logf("handling %s", req.RequestURI)

		backend.writeApiError(w, &opErrSomethingWentWrong)
	}
}

func (backend *opBackend) Serve(tb testing.TB) error {
	if backend.server != nil {
		return fmt.Errorf("server is already running")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", backend.FallbackHandler())
	mux.HandleFunc("/v1/vaults", backend.VaultsHandler())
	mux.HandleFunc(fmt.Sprintf("/v1/vaults/%s/items", mockVaultUUID), backend.ItemsHandler(mockVaultUUID))
	mux.HandleFunc(fmt.Sprintf("/v1/vaults/%s/items/%s", mockVaultUUID, mockItemUUID), backend.ItemHandler(mockVaultUUID, mockItemUUID))

	backend.server = &http.Server{
		Handler: mux,
	}

	return backend.server.Serve(backend.listener)
}

func (backend *opBackend) Shutdown(ctx context.Context) error {
	err := backend.server.Shutdown(ctx)
	if err == nil {
		backend.server = nil
	}
	return err
}

func (backend *opBackend) Close() error {
	err := backend.server.Close()
	if err == nil {
		backend.server = nil
	}
	return err
}

func TestMain(m *testing.M) {
	os.Setenv(EnvToken, mockToken)
	os.Setenv(EnvHost, mockHost)

	os.Exit(m.Run())
}
