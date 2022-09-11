package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/1Password/connect-sdk-go/onepassword"
	sm "github.com/cch123/supermonkey"
	_ "github.com/docker/go-plugins-helpers/sdk"
)

func tempFile(tb testing.TB, content string) *os.File {
	tb.Helper()

	fd, err := os.CreateTemp(tb.TempDir(), "test-op-secret-plugin-*")
	if err != nil {
		tb.Fatal(err)
	}

	_, err = fd.WriteString(content)
	if err != nil {
		tb.Fatal(err)
	}

	err = fd.Close()
	if err != nil {
		tb.Fatal(err)
	}

	return fd
}

type fullSocketAddressFunc = func(address string) (string, error)

func genFullSocketAddress(pluginSockDir string) fullSocketAddressFunc {
	return func(address string) (string, error) {
		if err := os.MkdirAll(pluginSockDir, 0755); err != nil {
			return "", err
		}
		if filepath.IsAbs(address) {
			return address, nil
		}
		return filepath.Join(pluginSockDir, address+".sock"), nil
	}
}

type newUnixListenerFunc = func(pluginName string, gid int) (net.Listener, string, error)

// genNewUnixListener replacement that doesn't try to chown the created socket,
// so the current user can remove the socket
// see github.com/docker/go-plugins-helpers/sdk.newUnixListener
func genNewUnixListener(tb testing.TB, fullSocketAddress fullSocketAddressFunc) newUnixListenerFunc {
	tb.Helper()

	return func(pluginName string, gid int) (net.Listener, string, error) {
		path, err := fullSocketAddress(pluginName)
		if err != nil {
			return nil, "", err
		}
		// listener, err := sockets.NewUnixSocket(path, gid)
		listener, err := newUnixSocket(path, gid)
		if err != nil {
			return nil, "", err
		}

		return listener, path, nil
	}
}

func newUnixSocket(path string, gid int) (net.Listener, error) {
	if err := syscall.Unlink(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	mask := syscall.Umask(0777)
	defer syscall.Umask(mask)

	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	// we don't chown as the test runs on temp, which has a sticky bit set
	// if err := os.Chown(path, 0, gid); err != nil {
	// 	l.Close()
	// 	return nil, err
	// }

	if err := os.Chmod(path, 0660); err != nil {
		l.Close()
		return nil, err
	}
	return l, nil
}

func Test_getConfig(t *testing.T) {
	originalHost := os.Getenv(EnvHost)
	defer os.Setenv(EnvHost, originalHost)

	originalTokenFile := os.Getenv(EnvTokenFile)
	defer os.Setenv(EnvTokenFile, originalTokenFile)

	originalToken := os.Getenv(EnvToken)
	defer os.Setenv(EnvToken, originalToken)

	testHostA := fmt.Sprintf("%s-%d", mockHost, rand.Uint64())
	testTokenA := fmt.Sprintf("%s-%d", mockToken, rand.Uint64())

	testHostB := fmt.Sprintf("%s-%d", mockHost, rand.Uint64())
	testTokenB := fmt.Sprintf("%s-%d", mockToken, rand.Uint64())
	testTokenFileB := tempFile(t, testTokenB)

	testTokenUnreadable := fmt.Sprintf("%s-%d", mockToken, rand.Uint64())
	testTokenFileUnreadable := tempFile(t, testTokenUnreadable)
	err := os.Chmod(testTokenFileUnreadable.Name(), 0220)
	if err != nil {
		t.Fatal(err)
	}

	testFileEmpty := tempFile(t, "")

	type EnvVar struct {
		set   bool
		name  string
		value string
	}
	tests := []struct {
		name    string
		vars    []EnvVar
		want    *config
		wantErr bool
	}{
		{
			name: "valid token",
			vars: []EnvVar{
				{true, EnvHost, testHostA},
				{true, EnvToken, testTokenA},
				{false, EnvTokenFile, ""},
			},
			want: &config{
				URL:   testHostA,
				Token: testTokenA,
			},
		},
		{
			name: "valid token file",
			vars: []EnvVar{
				{true, EnvHost, testHostB},
				{false, EnvToken, ""},
				{true, EnvTokenFile, testTokenFileB.Name()},
			},
			want: &config{
				URL:   testHostB,
				Token: testTokenB,
			},
		},
		{
			name: "env token over file token",
			vars: []EnvVar{
				{true, EnvHost, testHostB},
				{true, EnvToken, testTokenA},
				{true, EnvTokenFile, testTokenFileB.Name()},
			},
			want: &config{
				URL:   testHostB,
				Token: testTokenA,
			},
		},
		{
			name: "unset host",
			vars: []EnvVar{
				{false, EnvHost, ""},
				{false, EnvToken, ""},
				{false, EnvTokenFile, ""},
			},
			wantErr: true,
		},
		{
			name: "empty host",
			vars: []EnvVar{
				{true, EnvHost, ""},
				{false, EnvToken, ""},
				{false, EnvTokenFile, ""},
			},
			wantErr: true,
		},
		{
			name: "invalid host",
			vars: []EnvVar{
				{true, EnvHost, "lorem-ipsum"},
				{false, EnvToken, ""},
				{false, EnvTokenFile, ""},
			},
			wantErr: true,
		},
		{
			name: "invalid token file",
			vars: []EnvVar{
				{true, EnvHost, mockHost},
				{false, EnvToken, ""},
				{true, EnvTokenFile, t.TempDir()},
			},
			wantErr: true,
		},
		{
			name: "empty token file",
			vars: []EnvVar{
				{true, EnvHost, mockHost},
				{false, EnvToken, ""},
				{true, EnvTokenFile, ""},
			},
			wantErr: true,
		},
		{
			name: "unreadable token file",
			vars: []EnvVar{
				{true, EnvHost, mockHost},
				{false, EnvToken, ""},
				{true, EnvTokenFile, testTokenFileUnreadable.Name()},
			},
			wantErr: true,
		},
		{
			name: "zero token file",
			vars: []EnvVar{
				{true, EnvHost, mockHost},
				{false, EnvToken, ""},
				{true, EnvTokenFile, testFileEmpty.Name()},
			},
			wantErr: true,
		},
		{
			name: "invalid token file",
			vars: []EnvVar{
				{true, EnvHost, mockHost},
				{false, EnvToken, ""},
				{true, EnvTokenFile, "http://invalid"},
			},
			wantErr: true,
		},
		{
			name: "empty token",
			vars: []EnvVar{
				{true, EnvHost, mockHost},
				{true, EnvToken, ""},
				{false, EnvTokenFile, ""},
			},
			wantErr: true,
		},
		{
			name: "no token provided",
			vars: []EnvVar{
				{true, EnvHost, mockHost},
				{false, EnvToken, ""},
				{false, EnvTokenFile, ""},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, envVar := range tt.vars {
				if envVar.set == false {
					os.Unsetenv(envVar.name)
					continue
				}

				os.Setenv(envVar.name, envVar.value)
			}

			got, err := newConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("getConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_main(t *testing.T) {
	backend := newBackend(t, mockToken)
	defer backend.Close()

	socketDir := path.Join(t.TempDir(), "run/docker/plugins")

	fullSocketAddress := genFullSocketAddress(socketDir)
	newUnixListener := genNewUnixListener(t, fullSocketAddress)
	patch := sm.PatchByFullSymbolName("github.com/docker/go-plugins-helpers/sdk.newUnixListener", newUnixListener)
	defer patch.Unpatch()

	go main()

	<-time.After(time.Second)

	wantSocketPath := filepath.Join(socketDir, PluginName+".sock")
	info, err := os.Stat(wantSocketPath)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().IsRegular() {
		t.Fatalf("path %s is not a socket", wantSocketPath)
	}

	requestURL, err := url.ParseRequestURI(fmt.Sprintf("%s/v1/vaults", mockHost))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := backend.client.Do(&http.Request{
		Header: http.Header{
			"Authorization": []string{fmt.Sprintf("Bearer %s", mockToken)},
		},
		URL: requestURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatal(fmt.Errorf("response code %d, expected %d", resp.StatusCode, http.StatusOK))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	var got []onepassword.Vault
	err = json.Unmarshal(bodyBytes, &got)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, backend.vaults) {
		t.Fatalf("invalid response, got %#+v, wanted %#+v", got, backend.vaults)
	}
}
