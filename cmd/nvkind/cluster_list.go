package main

import (
	"fmt"
	"sort"

	"github.com/klueska/kind-with-gpus-examples/pkg/nvkind"
	"github.com/urfave/cli/v2"
)

func BuildClusterListCommand() *cli.Command {
	cmd := cli.Command{}
	cmd.Name = "list"
	cmd.Usage = "list all kind clusters (whether they have GPUs on them or not)"
	cmd.Action = runClusterList
	return &cmd
}

func runClusterList(c *cli.Context) error {
	clusters, err := nvkind.GetClusterNames()
	if err != nil {
		return fmt.Errorf("getting cluster names: %w", err)
	}

	clusterList := clusters.UnsortedList()
	sort.Strings(clusterList)

	if len(clusterList) == 0 {
		fmt.Println("No kind clusters found.")
	}

	for _, cluster := range clusterList {
		fmt.Println(cluster)
	}

	return nil
}
