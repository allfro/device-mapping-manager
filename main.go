package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-plugins-helpers/volume"
	_ "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

const pluginId = "dvd"

func main() {
	driver := DeviceVolumeDriver()
	handler := volume.NewHandler(driver)
	log.Println(handler.ServeUnix(pluginId, 0))
}

type deviceVolumeDriver struct {
	*client.Client
}

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
		filter := filters.NewArgs(filters.KeyValuePair{Key: "volume", Value: request.Name})

		containers, err := d.ContainerList(
			context.Background(),
			types.ContainerListOptions{Filters: filter},
		)

		if err != nil {
			log.Println(err)
			return
		} else if len(containers) == 0 {
			log.Println("aborting: could not find container that uses volume " + mountPoint.name)
			return
		}

		devicesAllowPath := path.Join("/sys/fs/cgroup/devices/docker", containers[0].ID, "devices.allow")

		if _, err := os.Stat(devicesAllowPath); os.IsNotExist(err) {
			//return nil, errors.New("could not find cgroup `devices.allow` file for specified container: " + devicesAllowPath)
			log.Println(errors.New("could not find cgroup `devices.allow` file for specified container: " + devicesAllowPath))
			return
		}

		var stat unix.Stat_t

		if err := unix.Stat(mountPoint.device, &stat); err != nil {
			//return nil, err
			log.Println(err)
			return
		}

		dev := uint64(stat.Rdev)
		input := fmt.Sprintf("c %d:%d rwm\n", unix.Major(dev), unix.Minor(dev))

		log.Println("Whitelisting `" + mountPoint.device + "` in `" + devicesAllowPath + "`")

		if err := os.WriteFile(devicesAllowPath, []byte(input), 0400); err != nil {
			//return nil, err
			log.Println(err)
			return
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

type pointer64 *int64

func DeviceVolumeDriver() *deviceVolumeDriver {
	cli, err := client.NewClientWithOpts(client.FromEnv)

	if err != nil {
		panic(err)
	}

	//m, err := cgroup2.LoadSystemd("/system.slice", "docker-9ac190cfc7040ffb1a56315b0c4aba9a554e72aa43164c4b94e84ee5ae3d07d9.scope")
	//
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//major := int64(10)
	//minor := int64(229)
	//err = m.Update(&cgroup2.Resources{
	//	Devices: []specs.LinuxDeviceCgroup{
	//		{
	//			Allow:  true,
	//			Type:   "c",
	//			Major:  &major,
	//			Minor:  &minor,
	//			Access: "rwm",
	//		},
	//	},
	//})
	//
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//os.Exit(0)
	return &deviceVolumeDriver{cli}
}
