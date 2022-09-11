package mount

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
)

type mountOptions map[string]string

func NewMountOptions(data []byte) (*mountOptions, error) {
	mntOptions := mountOptions{}
	err := mntOptions.UnmarshalText(data)
	return &mntOptions, err
}

func (mntOptions mountOptions) MarshalText() ([]byte, error) {
	options := make([]string, 0, len(mntOptions))

	for key, value := range mntOptions {
		if value == "" {
			options = append(options, key)
		} else {
			options = append(options, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return []byte(strings.Join(options, ",")), nil
}

func (mntOptions mountOptions) UnmarshalText(text []byte) error {
	entries := bytes.Split(text, []byte(","))

	for _, entry := range entries {
		parts := bytes.SplitN(entry, []byte("="), 2)
		key := string(parts[0])
		switch len(parts) {
		case 1:
			mntOptions[key] = ""
		case 2:
			mntOptions[key] = string(parts[1])
		default:
			return fmt.Errorf("failed to split mount options")
		}
	}

	return nil
}

func (mntOptions mountOptions) String() string {
	data, err := mntOptions.MarshalText()
	if err != nil {
		panic(err)
	}

	return string(data)
}

type procMount struct {
	device  string
	mount   string
	fsType  string
	options *mountOptions
	dump    int
	pass    int
}

func NewProcMount(data []byte) (*procMount, error) {
	mount := procMount{}
	err := mount.UnmarshalText(data)
	return &mount, err
}

func (mnt *procMount) MarshalText() ([]byte, error) {
	return nil, nil
}

func (mnt *procMount) UnmarshalText(text []byte) error {
	var options string
	_, err := fmt.Sscanf(string(text), "%s %s %s %s %d %d", &mnt.device, &mnt.mount, &mnt.fsType, &options, &mnt.dump, &mnt.pass)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to parse mount: %w", err)
	}

	mnt.options, err = NewMountOptions([]byte(options))
	if err != nil {
		return fmt.Errorf("failed to parse mount options: %w", err)
	}

	return nil
}

func (mnt *procMount) String() string {
	data, err := mnt.MarshalText()
	if err != nil {
		panic(err)
	}

	return string(data)
}

type mountsCache struct {
	// io.ReaderFrom

	mounts   map[string]*procMount
	checksum [sha256.Size]byte
}

func NewMountsCache(r io.Reader) (*mountsCache, error) {
	mntReader := mountsCache{}
	_, err := mntReader.ReadFrom(r)
	return &mntReader, err
}

func (mounts *mountsCache) ReadFrom(r io.Reader) (n int64, err error) {
	data, err := io.ReadAll(r)
	if err != nil && err != io.EOF {
		return 0, err
	}

	checksum := sha256.Sum256(data)
	if mounts.checksum == checksum {
		return 0, nil
	}

	lines := bytes.Split(data, []byte("\n"))

	for _, line := range lines {
		mount, err := NewProcMount(line)
		if err != nil {
			return 0, fmt.Errorf("failed to build new proc mount: %w", err)
		}

		mounts.mounts[mount.mount] = mount
	}

	mounts.checksum = checksum
	return int64(len(data)), nil
}
