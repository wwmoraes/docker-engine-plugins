package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/mitchellh/mapstructure"
	bolt "go.etcd.io/bbolt"
)

var volumeBucket = []byte("volumes")

type cifsDriver struct {
	db *bolt.DB
}

type Options map[string]string

func (options Options) String() string {
	entries := make([]string, 0, len(options))

	for key, value := range options {
		if value == "" {
			entries = append(entries, key)
		} else {
			entries = append(entries, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return strings.Join(entries, ",")
}

type Status struct {
	Mounted bool
	Share   string
	Options Options
}

func NewDriver() (volume.Driver, error) {
	db, err := bolt.Open("cifs.db", 0640, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(volumeBucket)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &cifsDriver{
		db: db,
	}, nil
}

func (driver *cifsDriver) getVolume(name string) (info *volume.Volume, err error) {
	err = driver.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(volumeBucket)
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", string(volumeBucket))
		}

		value := bucket.Get([]byte(name))
		if value == nil {
			return fmt.Errorf("volume %s does not exist", name)
		}

		return gob.NewDecoder(bytes.NewReader(value)).Decode(&info)
	})

	return info, err
}

func (driver *cifsDriver) putVolume(info *volume.Volume) error {
	var data bytes.Buffer
	err := gob.NewEncoder(&data).Encode(info)
	if err != nil {
		return err
	}

	return driver.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(volumeBucket)
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", string(volumeBucket))
		}

		return bucket.Put([]byte(info.Name), data.Bytes())
	})
}

func (driver *cifsDriver) deleteVolume(name string) error {
	return driver.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(volumeBucket)
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", string(volumeBucket))
		}

		return bucket.Delete([]byte(name))
	})
}

func (driver *cifsDriver) Create(req *volume.CreateRequest) error {
	share, exists := req.Options["share"]
	if !exists || share == "" {
		return fmt.Errorf("share must be provided")
	}
	delete(req.Options, "share")

	statusData := make(map[string]interface{})
	err := mapstructure.Decode(Status{
		Mounted: false,
		Share:   share,
		Options: req.Options,
	}, &statusData)
	if err != nil {
		return err
	}

	return driver.putVolume(&volume.Volume{
		Name:       req.Name,
		Mountpoint: "",
		CreatedAt:  time.Now().String(),
		Status:     statusData,
	})
}

func (driver *cifsDriver) List() (response *volume.ListResponse, err error) {
	response = &volume.ListResponse{}

	err = driver.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(volumeBucket)
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", string(volumeBucket))
		}

		response.Volumes = make([]*volume.Volume, 0, bucket.Stats().KeyN)

		return bucket.ForEach(func(k, v []byte) error {
			var info volume.Volume

			err = gob.NewDecoder(bytes.NewReader(v)).Decode(&info)
			if err != nil {
				return err
			}

			response.Volumes = append(response.Volumes, &info)

			return nil
		})
	})

	return response, err
}

func (driver *cifsDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	info, err := driver.getVolume(req.Name)

	return &volume.GetResponse{
		Volume: info,
	}, err
}

func (driver *cifsDriver) Remove(req *volume.RemoveRequest) error {
	return driver.deleteVolume(req.Name)
}

func (driver *cifsDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	info, err := driver.getVolume(req.Name)
	if err != nil {
		return nil, err
	}

	return &volume.PathResponse{
		Mountpoint: info.Mountpoint,
	}, nil
}

func (driver *cifsDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	info, err := driver.getVolume(req.Name)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("volume %s not found", req.Name)
	}

	var status Status
	err = mapstructure.Decode(info.Status, &status)
	if err != nil {
		return nil, err
	}

	if status.Mounted {
		return nil, fmt.Errorf("volume %s is already mounted", req.Name)
	}

	info.Mountpoint = path.Join(volume.DefaultDockerRootDirectory, req.ID)

	cmd := exec.Command("mount", "-t", "cifs", "-o", status.Options.String(), status.Share, info.Mountpoint)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	status.Mounted = true

	err = mapstructure.Decode(status, &info.Status)
	if err != nil {
		return nil, err
	}

	err = driver.putVolume(info)

	return &volume.MountResponse{
		Mountpoint: info.Mountpoint,
	}, err
}

func (driver *cifsDriver) Unmount(req *volume.UnmountRequest) error {
	info, err := driver.getVolume(req.Name)
	if err != nil {
		return err
	}

	if info == nil {
		return fmt.Errorf("volume %s not found", req.Name)
	}

	mountPoint := path.Join(volume.DefaultDockerRootDirectory, req.ID)
	if info.Mountpoint != mountPoint {
		return fmt.Errorf("volume %s mount point does not match", req.Name)
	}

	status := Status{}
	err = mapstructure.Decode(info.Status, &status)
	if err != nil {
		return err
	}

	if !status.Mounted {
		return fmt.Errorf("volume %s is not mounted", req.Name)
	}

	cmd := exec.Command("umount", info.Mountpoint)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return err
	}

	info.Mountpoint = ""

	return driver.putVolume(info)
}

func (driver *cifsDriver) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}
