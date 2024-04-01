/*
 * Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"k8s.io/klog/v2"
)

// Version is the version of this CLI (overwritable at build time)
var Version = "devel"

func main() {
	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "nvkind"
	c.Usage = "kind for use with NVIDIA GPUs"
	c.Version = Version
	c.EnableBashCompletion = true

	// Register the subcommands with the top-level CLI
	c.Commands = []*cli.Command{
		BuildClusterCommand(),
	}

	// Run the CLI
	err := c.Run(os.Args)
	if err != nil {
		klog.Fatalf("Error: %v", err)
	}
}
