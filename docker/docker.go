package docker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/go-connections/nat"

	"github.com/codegangsta/cli"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Run spins up a docker container
func Run(img, port, name string, c *cli.Context) (string, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	r, err := cli.ImagePull(ctx, img, types.ImagePullOptions{})
	defer r.Close()
	if err != nil {
		return "", err
	}

	tcp := nat.Port(fmt.Sprintf("%s/tcp", port))

	hostConfig := container.HostConfig{
		PortBindings: nat.PortMap{
			tcp: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port,
				},
			},
		},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        img,
		ExposedPorts: nat.PortSet{tcp: struct{}{}},
	}, &hostConfig, nil, name)
	if err != nil {
		return "", err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	iChan := make(chan os.Signal, 2)
	signal.Notify(iChan, os.Interrupt, syscall.SIGTERM)
	go func(ID string) {
		<-iChan
		log.Printf("Killing %s... \n", name)
		if err := Stop(ID); err != nil {
			log.Printf("Failed killing %s\n", name)
			os.Exit(1)
		}
		log.Printf("%s stopped and removed\n", name)
		os.Exit(0)
	}(resp.ID)

	return resp.ID, nil
}

// Stop stops and removes a container
func Stop(ID string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	return cli.ContainerRemove(context.Background(), ID, types.ContainerRemoveOptions{Force: true})
}
