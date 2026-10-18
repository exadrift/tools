package talossdk

import (
	"context"
	"fmt"
	"time"

	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/siderolabs/talos/pkg/machinery/api/cluster"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/api/storage"
	"github.com/siderolabs/talos/pkg/machinery/client"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
)

func NewMaintenanceClient(cfg *config.Config) (*TalosClient, error) {
	c, err := client.New(context.Background(),
		client.WithEndpoints(cfg.IpAddress),
		client.WithMaintenanceMode(cfg.IpAddress, nil),
	)
	if err != nil {
		return nil, err
	}

	return &TalosClient{
		client: c,
	}, nil
}

func NewAuthenticatedClient(machineConfig *config.MachineConfig) (*TalosClient, error) {
	c, err := client.New(context.Background(),
		client.WithEndpoints(machineConfig.IpAddress),
		client.WithConfig(machineConfig.TalosConfig),
	)
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

func (t *TalosClient) Disks(ctx context.Context) (*storage.DisksResponse, error) {
	return t.client.Disks(ctx)
}

func (t *TalosClient) GetVersion(ctx context.Context) (*talosconfig.VersionContract, error) {
	resp, err := t.client.Version(ctx)
	if err != nil {
		return nil, err
	}

	vers := resp.GetMessages()
	if len(vers) == 0 {
		return nil, fmt.Errorf("no version information was returned")
	}

	tag := vers[0].GetVersion().Tag
	if tag == "" {
		return nil, fmt.Errorf("version tag returned was empty")
	}

	versionContract, err := talosconfig.ParseContractFromVersion(tag)
	if err != nil {
		return nil, fmt.Errorf("unable to parse talos version: %w", err)
	}

	return versionContract, nil
}

func (t *TalosClient) ApplyConfiguration(ctx context.Context, configData []byte, dryRun bool) error {
	_, err := t.client.ApplyConfiguration(ctx, &machine.ApplyConfigurationRequest{
		Data:   configData,
		Mode:   machine.ApplyConfigurationRequest_AUTO,
		DryRun: dryRun,
	})
	if err != nil {
		return fmt.Errorf("unable to apply machine config: %w", err)
	}

	return nil
}

func (t *TalosClient) Bootstrap(ctx context.Context) error {
	return t.client.Bootstrap(ctx, &machine.BootstrapRequest{})
}

func (t *TalosClient) HealthCheck(ctx context.Context, cfg *config.Config) (cluster.ClusterService_HealthCheckClient, error) {
	return t.client.ClusterHealthCheck(ctx, time.Minute*15, &cluster.ClusterInfo{
		ControlPlaneNodes: []string{cfg.IpAddress},
	})
}

func (t *TalosClient) KubeConfig(ctx context.Context) ([]byte, error) {
	return t.client.Kubeconfig(ctx)
}
