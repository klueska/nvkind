package main

import (
	"github.com/urfave/cli/v2"
)

func BuildClusterCommand() *cli.Command {
	cmd := cli.Command{}
	cmd.Name = "cluster"
	cmd.Usage = "perform operations on cluster with NVIDIA GPUs"
	cmd.Subcommands = []*cli.Command{
		BuildClusterListCommand(),
		BuildClusterCreateCommand(),
		BuildClusterPrintGPUsCommand(),
	}
	return &cmd
}
