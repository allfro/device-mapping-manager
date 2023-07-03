//go:build linux

package main

// #include "ctypes.h"
import "C"
import (
	"context"
	"device-volume-driver/internal/cgroup"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	_ "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const pluginId = "dvd"
const rootPath = "/host"

func Ptr[T any](v T) *T {
	return &v
}

func main() {
	listenForMounts()
}

func getDeviceInfo(devicePath string) (string, int64, int64, error) {
	var stat unix.Stat_t

	if err := unix.Lstat(devicePath, &stat); err != nil {
		log.Println(err)
		return "", -1, -1, err
	}

	var deviceType string

	switch stat.Mode & unix.S_IFMT {
	case unix.S_IFBLK:
		deviceType = "b"
	case unix.S_IFCHR:
		deviceType = "c"
	default:
		log.Println("aborting: device is neither a character or block device")
		return "", -1, -1, fmt.Errorf("unsupported device type... aborting")
	}

	major := int64(unix.Major(stat.Rdev))
	minor := int64(unix.Minor(stat.Rdev))

	log.Printf("Found device: %s %s %d:%d\n", devicePath, deviceType, major, minor)

	return deviceType, major, minor, nil
}

func listenForMounts() {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		log.Fatal(err)
	}

	defer cli.Close()

	msgs, errs := cli.Events(
		ctx,
		types.EventsOptions{Filters: filters.NewArgs(filters.Arg("event", "start"))},
	)

	for {
		select {
		case err := <-errs:
			log.Fatal(err)
		case msg := <-msgs:
			info, err := cli.ContainerInspect(ctx, msg.Actor.ID)

			if err != nil {
				panic(err)
			} else {
				pid := info.State.Pid
				version, err := cgroup.GetDeviceCGroupVersion("/", pid)

				log.Printf("The cgroup version for process %d is: %v\n", pid, version)

				if err != nil {
					log.Println(err)
					break
				}

				log.Printf("Checking mounts for process %d\n", pid)

				for _, mount := range info.Mounts {
					log.Printf(
						"%s/%v requested a volume mount for %s at %s\n",
						msg.Actor.ID, info.State.Pid, mount.Source, mount.Destination,
					)

					if !strings.HasPrefix(mount.Source, "/dev") {
						log.Printf("%s is not a device... skipping\n", mount.Source)
						continue
					}

					api, err := cgroup.New(version)
					cgroupPath, sysfsPath, err := api.GetDeviceCGroupMountPath("/", pid)

					if err != nil {
						log.Println(err)
						break
					}

					cgroupPath = path.Join(rootPath, sysfsPath, cgroupPath)

					log.Printf("The cgroup path for process %d is at %v\n", pid, cgroupPath)

					if fileInfo, err := os.Stat(mount.Source); err != nil {
						log.Println(err)
						continue
					} else {
						if fileInfo.IsDir() {
							err := filepath.Walk(mount.Source,
								func(path string, info os.FileInfo, err error) error {
									if err != nil {
										return err
									} else if info.IsDir() {
										return nil
									} else if err = applyDeviceRules(api, path, cgroupPath, pid); err != nil {
										log.Println(err)
									}
									return nil
								})
							if err != nil {
								log.Println(err)
							}
						} else {
							if err = applyDeviceRules(api, mount.Source, cgroupPath, pid); err != nil {
								log.Println(err)
							}
						}
					}

				}
			}
		}
	}
}

func applyDeviceRules(api cgroup.Interface, mountPath string, cgroupPath string, pid int) error {
	deviceType, major, minor, err := getDeviceInfo(mountPath)

	if err != nil {
		log.Println(err)
		return err
	} else {
		log.Printf("Adding device rule for process %d at %s\n", pid, cgroupPath)
		err = api.AddDeviceRules(cgroupPath, []cgroup.DeviceRule{
			{
				Access: "rwm",
				Major:  Ptr[int64](major),
				Minor:  Ptr[int64](minor),
				Type:   deviceType,
				Allow:  true,
			},
		})

		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}
