package talossdk

import (
	"context"
	"fmt"

	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/siderolabs/talos/pkg/machinery/client"
)

func NewClient(cfg *config.Config, maintenanceMode bool) (*TalosClient, error) {
	var opts = []client.OptionFunc{
		client.WithEndpoints(cfg.IpAddress),
	}
	if maintenanceMode {
		opts = append(opts, client.WithMaintenanceMode(cfg.IpAddress, nil))
	}
	c, err := client.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	return &TalosClient{
		client: c,
	}, nil
}

type TalosClient struct {
	client *client.Client
}

func (t *TalosClient) Close() error {
	return t.client.Close()
}

func (t *TalosClient) Probe(ctx context.Context) error {
	_, err := t.client.Disks(ctx)
	if err != nil {
		return fmt.Errorf("probe failed")
	}

	return nil
}
