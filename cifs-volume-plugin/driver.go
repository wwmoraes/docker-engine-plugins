package main

import (
	volumeplugin "github.com/docker/go-plugins-helpers/volume"
)

type cifsDriver struct {
}

func (driver *cifsDriver) Create(req *volumeplugin.CreateRequest) error {
	return nil
}

func (driver *cifsDriver) List() (*volumeplugin.ListResponse, error) {
	return nil, nil
}

func (driver *cifsDriver) Get(req *volumeplugin.GetRequest) (*volumeplugin.GetResponse, error) {
	return nil, nil
}

func (driver *cifsDriver) Remove(req *volumeplugin.RemoveRequest) error {
	return nil
}

func (driver *cifsDriver) Path(req *volumeplugin.PathRequest) (*volumeplugin.PathResponse, error) {
	return nil, nil
}

func (driver *cifsDriver) Mount(req *volumeplugin.MountRequest) (*volumeplugin.MountResponse, error) {
	return nil, nil
}

func (driver *cifsDriver) Unmount(req *volumeplugin.UnmountRequest) error {
	return nil
}

func (driver *cifsDriver) Capabilities() *volumeplugin.CapabilitiesResponse {
	return &volumeplugin.CapabilitiesResponse{
		Capabilities: volumeplugin.Capability{
			Scope: "local",
		},
	}
}
