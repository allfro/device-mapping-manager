package main

import (
	"errors"
	"fmt"
	"github.com/docker/go-plugins-helpers/volume"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

const pluginId = "device-volume-driver"

func main() {
	driver := DeviceVolumeDriver()
	handler := volume.NewHandler(driver)
	log.Println(handler.ServeUnix(pluginId, 0))
}

type deviceVolumeDriver struct{}

type mountPoint struct {
	name   string
	device string
}

var mountPoints = make(map[string]mountPoint)

func (d deviceVolumeDriver) Create(request *volume.CreateRequest) error {
	device, ok := request.Options["device"]

	if !ok {
		return errors.New("you must specify the `device` option")
	}

	mountPoints[request.Name] = mountPoint{
		name:   strings.Clone(request.Name),
		device: strings.Clone(device),
	}

	return nil
}

func (d deviceVolumeDriver) List() (*volume.ListResponse, error) {
	var volumes []*volume.Volume

	for _, mountPoint := range mountPoints {
		volumes = append(volumes, &volume.Volume{Name: mountPoint.name, Mountpoint: mountPoint.device})
	}

	return &volume.ListResponse{
		Volumes: volumes,
	}, nil
}

func (d deviceVolumeDriver) Get(request *volume.GetRequest) (*volume.GetResponse, error) {
	mountPoint, ok := mountPoints[request.Name]

	if !ok {
		return nil, errors.New("no such volume")
	}

	return &volume.GetResponse{Volume: &volume.Volume{Name: mountPoint.name, Mountpoint: mountPoint.device}}, nil
}

func (d deviceVolumeDriver) Remove(request *volume.RemoveRequest) error {
	delete(mountPoints, request.Name)
	return nil
}

func (d deviceVolumeDriver) Path(request *volume.PathRequest) (*volume.PathResponse, error) {
	mountPoint, ok := mountPoints[request.Name]
	if !ok {
		return nil, errors.New("no such volume")
	}
	return &volume.PathResponse{Mountpoint: mountPoint.device}, nil
}

func (d deviceVolumeDriver) Mount(request *volume.MountRequest) (*volume.MountResponse, error) {
	mountPoint, ok := mountPoints[request.Name]

	if !ok {
		return nil, errors.New("mountpoint does not exist")
	}

	go func() {
		time.Sleep(time.Second * 1)
		devicesAllowPath := path.Join("/sys/fs/cgroup/devices/docker", request.ID, "devices.allow")

		if _, err := os.Stat(devicesAllowPath); os.IsNotExist(err) {
			//return nil, errors.New("could not find cgroup `devices.allow` file for specified container: " + devicesAllowPath)
			log.Println(errors.New("could not find cgroup `devices.allow` file for specified container: " + devicesAllowPath))
		}

		var stat unix.Stat_t

		if err := unix.Stat(mountPoint.device, &stat); err != nil {
			//return nil, err
			log.Println(err)
		}

		dev := uint64(stat.Rdev)
		input := fmt.Sprintf("c %i:%i rwm\n", unix.Major(dev), unix.Minor(dev))

		if err := os.WriteFile(devicesAllowPath, []byte(input), 0400); err != nil {
			//return nil, err
			log.Println(err)
		}
	}()

	return &volume.MountResponse{Mountpoint: mountPoint.device}, nil
}

func (d deviceVolumeDriver) Unmount(request *volume.UnmountRequest) error {
	return nil
}

func (d deviceVolumeDriver) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "local"}}
}

func DeviceVolumeDriver() *deviceVolumeDriver {
	return &deviceVolumeDriver{}
}
