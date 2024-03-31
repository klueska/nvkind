package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/klueska/kind-with-gpus-examples/pkg/nvkind"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ClusterListFlags struct {
	Name       string
	KubeConfig string
}

func BuildClusterListCommand() *cli.Command {
	flags := ClusterListFlags{}

	cmd := cli.Command{}
	cmd.Name = "list"
	cmd.Usage = "list all kind clusters (whether they have GPUs on them or not)"
	cmd.Action = func(ctx *cli.Context) error {
		return runClusterList(ctx, &flags)
	}

	cmd.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "name",
			Usage:       "the name of the cluster to print GPUs for",
			Destination: &flags.Name,
			EnvVars:     []string{"KIND_CLUSTER_NAME"},
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

func runClusterList(c *cli.Context, f *ClusterListFlags) error {
	if err := f.updateFlagsWithDefaults(); err != nil {
		return fmt.Errorf("updating flags with defaults: %w", err)
	}

	clusters, err := nvkind.GetClusterNames()
	if err != nil {
		return fmt.Errorf("getting cluster names: %w", err)
	}

	clusterList := clusters.UnsortedList()
	sort.Strings(clusterList)

	for _, cluster := range clusterList {
		fmt.Println(cluster)
	}

	return nil
}

func (f *ClusterListFlags) updateFlagsWithDefaults() error {
	if f.KubeConfig == "" {
		if home := homedir.HomeDir(); home != "" {
			f.KubeConfig = home + "/.kube/config"
		}
	}

	if f.Name != "" {
		return nil
	}

	config, err := clientcmd.LoadFromFile(f.KubeConfig)
	if err != nil {
		return fmt.Errorf("marshaling GPU info: %w", err)
	}

	if config.CurrentContext == "" {
		return fmt.Errorf("no current kubecontext set")
	}

	if !strings.HasPrefix(config.CurrentContext, "kind-") {
		return fmt.Errorf("current kubecontext is not a kind cluster: %v", config.CurrentContext)
	}

	f.Name = strings.TrimPrefix(config.CurrentContext, "kind-")

	return nil
}
