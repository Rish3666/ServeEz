package container

import (
	"context"

	"github.com/Rish3666/ServeEz/internal/api"
)

type Manager interface {
	Create(ctx context.Context, workload string, spec api.WorkloadSpec, replica int) (api.ContainerStatus, error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	Restart(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	Inspect(ctx context.Context, id string) (api.ContainerStatus, error)
	List(ctx context.Context, workload string) ([]api.ContainerStatus, error)
}

type DockerConfig struct {
	Host    string
	Version string
}
