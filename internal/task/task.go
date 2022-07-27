package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type State int

const (
	Pending State = iota
	Scheduled
	Completed
	Running
	Failed
)

var stateTransitionMap = map[State][]State{
	Pending:   {Scheduled},
	Scheduled: {Scheduled, Running, Failed},
	Running:   {Running, Completed, Failed},
	Completed: {},
	Failed:    {},
}

func Contains(states []State, state State) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

func ValidStateTransition(src State, dst State) bool {
	return Contains(stateTransitionMap[src], dst)
}

type Task struct {
	ID            uuid.UUID
	Name          string
	State         State
	Image         string
	Memory        int64
	Disk          int64
	ExposedPorts  nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string
	StartTime     time.Time
	FinishTime    time.Time
	ContainerID   string
}
type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}

type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	Cmd           []string
	Image         string
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy string
}

type Docker struct {
	Client      *client.Client
	Config      Config
	ContainerID string
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerID string
	Result      string
}

func NewConfig(t *Task) *Config {
	return &Config{
		Name:   t.Name,
		Image:  t.Image,
		Memory: t.Memory,
		Disk:   t.Disk,
	}
}

func NewDocker(c *Config) *Docker {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		panic(err)
	}
	return &Docker{
		Client:      cli,
		Config:      *c,
		ContainerID: c.Name,
	}
}

//nolint:funlen
func (d *Docker) Run() DockerResult {
	// Pull Image
	ctx := context.Background()
	reader, err := d.Client.ImagePull(
		ctx, d.Config.Image, types.ImagePullOptions{})
	if err != nil {
		fmt.Printf("Error pulling image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		fmt.Printf("Error pulling image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	// Create Container
	rp := container.RestartPolicy{
		Name: d.Config.RestartPolicy,
	}
	r := container.Resources{
		Memory: d.Config.Memory,
	}
	cc := container.Config{
		Image: d.Config.Image,
		Env:   d.Config.Env,
	}
	hc := container.HostConfig{
		RestartPolicy:   rp,
		Resources:       r,
		PublishAllPorts: true,
	}

	resp, err := d.Client.ContainerCreate(
		ctx, &cc, &hc, nil, nil, d.Config.Name)
	if err != nil {
		fmt.Printf("Error creating container using image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	// start container
	err2 := d.Client.ContainerStart(
		ctx, resp.ID, types.ContainerStartOptions{})
	if err2 != nil {
		fmt.Printf("Erroßr starting container %s: %v\n", resp.ID, err2)
		return DockerResult{Error: err2}
	}

	d.ContainerID = resp.ID
	out, err := d.Client.ContainerLogs(
		ctx,
		resp.ID,
		types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		fmt.Printf("Error getting logs for container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	if err != nil {
		fmt.Printf("Error getting logs for container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}
	return DockerResult{
		ContainerID: resp.ID,
		Action:      "start",
		Result:      "success",
	}
}

func (d *Docker) Stop() DockerResult {
	ctx := context.Background()
	fmt.Printf(
		"Attempting to stop container %v", d.ContainerID)
	err := d.Client.ContainerStop(ctx, d.ContainerID, nil)
	if err != nil {
		panic(err)
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	}

	err = d.Client.ContainerRemove(
		ctx,
		d.ContainerID,
		removeOptions,
	)
	if err != nil {
		panic(err)
	}
	return DockerResult{Action: "stop", Result: "success", Error: nil}
}
