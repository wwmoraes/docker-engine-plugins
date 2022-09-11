package main

import (
	volumeplugin "github.com/docker/go-plugins-helpers/volume"
)

func NewHandler() *volumeplugin.Handler {
	return volumeplugin.NewHandler(&cifsDriver{})
}
