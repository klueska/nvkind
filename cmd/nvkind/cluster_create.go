package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/klueska/kind-with-gpus-examples/pkg/nvkind"
	"github.com/urfave/cli/v2"
)

type ClusterCreateFlags struct {
	Name           string
	ConfigTemplate string
	ConfigValues   string
	KubeConfig     string
}

func BuildClusterCreateCommand() *cli.Command {
	flags := ClusterCreateFlags{}

	cmd := cli.Command{}
	cmd.Name = "create"
	cmd.Usage = "create a cluster with support for NVIDIA GPUs"
	cmd.Action = func(ctx *cli.Context) error {
		return runClusterCreate(ctx, &flags)
	}

	cmd.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "name",
			Usage:       "the name of the cluster to create (default nvkind-<random>)",
			Destination: &flags.Name,
			EnvVars:     []string{"KIND_CLUSTER_NAME"},
		},
		&cli.StringFlag{
			Name:        "config-template",
			Usage:       "the path to a custom kind config template",
			Destination: &flags.ConfigTemplate,
			EnvVars:     []string{"KIND_CLUSTER_CONFIG_TEMPLATE"},
		},
		&cli.StringFlag{
			Name:        "config-values",
			Usage:       "the path to a values file to fill in the variables from a kind config template",
			Destination: &flags.ConfigValues,
			EnvVars:     []string{"KIND_CLUSTER_CONFIG_VALUES"},
		},
		&cli.StringFlag{
			Name:        "kubeconfig",
			Usage:       "Absolute path to the `KUBECONFIG` file. Either this flag or the KUBECONFIG env variable need to be set if the driver is being run out of cluster.",
			Destination: &flags.KubeConfig,
			EnvVars:     []string{"KUBECONFIG"},
		},
	}

	return &cmd
}

func runClusterCreate(c *cli.Context, f *ClusterCreateFlags) error {
	clusterOptions, err := f.gatherClusterOptions()
	if err != nil {
		return fmt.Errorf("gathering cluster options: %w", err)
	}

	cluster, err := nvkind.NewCluster(clusterOptions...)
	if err != nil {
		return fmt.Errorf("new cluster: %w", err)
	}

	if err := cluster.Create(); err != nil {
		return fmt.Errorf("creating cluster: %w", err)
	}

	nodes, err := cluster.GetNodes()
	if err != nil {
		return fmt.Errorf("getting cluster nodes: %w", err)
	}

	for _, node := range nodes {
		if !node.HasGPUs() {
			continue
		}
		if err := node.InstallContainerToolkit(); err != nil {
			return fmt.Errorf("installing container toolkit on node '%v': %w", node.Name, err)
		}
		if err := node.ConfigureContainerRuntime(); err != nil {
			return fmt.Errorf("configuring container runtime on node '%v': %w", node.Name, err)
		}
		if err := node.PatchProcDriverNvidia(); err != nil {
			return fmt.Errorf("patching /proc/driver/nvidia on node '%v': %w", node.Name, err)
		}
	}

	return nil
}

func (f *ClusterCreateFlags) gatherConfigOptions() ([]nvkind.ConfigOption, error) {
	var configOptions []nvkind.ConfigOption

	if f.ConfigTemplate != "" {
		configOptions = append(configOptions, nvkind.WithConfigTemplate(f.ConfigTemplate))
	}

	if f.ConfigValues != "" {
		var err error
		var configValues []byte

		if f.ConfigValues == "-" {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				configValues = append(configValues, scanner.Bytes()...)
				configValues = append(configValues, '\n')
			}
		} else {
			configValues, err = os.ReadFile(f.ConfigValues)
			if err != nil {
				return nil, fmt.Errorf("reading file: %w", err)
			}
		}

		configOptions = append(configOptions, nvkind.WithConfigValues(configValues))
	}

	return configOptions, nil
}

func (f *ClusterCreateFlags) gatherClusterOptions() ([]nvkind.ClusterOption, error) {
	var clusterOptions []nvkind.ClusterOption

	if f.Name != "" {
		clusterOptions = append(clusterOptions, nvkind.WithName(f.Name))
	}

	if f.KubeConfig != "" {
		clusterOptions = append(clusterOptions, nvkind.WithKubeConfig(f.KubeConfig))
	}

	configOptions, err := f.gatherConfigOptions()
	if err != nil {
		return nil, fmt.Errorf("gathering config options: %w", err)
	}

	if len(configOptions) != 0 {
		config, err := nvkind.NewConfig(configOptions...)
		if err != nil {
			return nil, fmt.Errorf("new config: %w", err)
		}
		clusterOptions = append(clusterOptions, nvkind.WithConfig(config))
	}

	return clusterOptions, nil
}
