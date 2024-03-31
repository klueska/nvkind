#!/usr/bin/env bash

# Copyright 2024 NVIDIA CORPORATION.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# A reference to the current directory where this script is located
CURRENT_DIR="$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)"

set -ex
set -o pipefail

source "${CURRENT_DIR}/common.sh"

NUM_WORKERS="${#}"
: ${CLUSTER_NAME:="explicit-gpus-$(tr -dc a-z0-9 </dev/urandom | head -c 5)"}
: ${CLUSTER_CONFIG_PATH:=${DEFAULT_CLUSTER_CONFIG_PATH}}

create_cluster ${KIND_IMAGE} ${CLUSTER_NAME} ${CLUSTER_CONFIG_PATH} ${NUM_WORKERS}

for worker_id in $(seq ${NUM_WORKERS}); do
	allowed_gpus="${!worker_id}"
	install_container_toolkit ${CLUSTER_NAME} ${worker_id}
	configure_container_runtime ${CLUSTER_NAME} ${worker_id}
	patch_proc_driver_nvidia ${CLUSTER_NAME} ${NUM_GPUS} ${worker_id} "${allowed_gpus}"
done
