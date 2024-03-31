# Examples of how to run `kind` with GPUs

This repo provides examples / tooling to run `kind` clusters with access to GPUs.

Unfortunately, running `kind` with access to GPUs is not very straightforward.
There is no standard way to inject GPUs support into a `kind` worker node, and
even with a series of "hacks" to make it possible, some post processing still
needs to be performed to ensure that different sets of GPUs can be isolated to
different worker nodes.

The scripts contained in this repository encapsulate the set of steps required
to do what is described above. They can either be run directly, or you can use
them as a starting point to write your own.

## Prerequisites

The following prerequisites are required to run run the scripts provided by this repo:

    Prerequisite | Link
    ------------ | -------------------------------------
    jq           | https://jqlang.github.io/jq/download/
    docker       | https://docs.docker.com/get-docker/
    kind         | https://kind.sigs.k8s.io/docs/user/quick-start/#installation
    kubectl      | https://kubernetes.io/docs/tasks/tools/
    helm         | https://helm.sh/docs/intro/install/

You must also ensure that you are running on a host with a working NVIDIA
driver and an `nvidia-container-toolkit` configured for use with `docker`.

    Prerequisite             | Link
    ------------------------ | -------------------------------------
    nvidia-driver            | https://www.nvidia.com/download/index.aspx
    nvidia-container-toolkit | https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html

Running `nvidia-smi -L` on a host with a functioning driver should produce
output such as the following:

```bash
$ nvidia-smi -L
GPU 0: NVIDIA A100-SXM4-40GB (UUID: GPU-4cf8db2d-06c0-7d70-1a51-e59b25b2c16c)
GPU 1: NVIDIA A100-SXM4-40GB (UUID: GPU-4404041a-04cf-1ccf-9e70-f139a9b1e23c)
GPU 2: NVIDIA A100-SXM4-40GB (UUID: GPU-79a2ba02-a537-ccbf-2965-8e9d90c0bd54)
GPU 3: NVIDIA A100-SXM4-40GB (UUID: GPU-662077db-fa3f-0d8f-9502-21ab0ef058a2)
GPU 4: NVIDIA A100-SXM4-40GB (UUID: GPU-ec9d53cc-125d-d4a3-9687-304df8eb4749)
GPU 5: NVIDIA A100-SXM4-40GB (UUID: GPU-3eb87630-93d5-b2b6-b8ff-9b359caf4ee2)
GPU 6: NVIDIA A100-SXM4-40GB (UUID: GPU-8216274a-c05d-def0-af18-c74647300267)
GPU 7: NVIDIA A100-SXM4-40GB (UUID: GPU-b1028956-cfa2-0990-bf4a-5da9abb51763)
```

Likewise, running the following on a host with a functioning
`nvidia-container-toolkit` that has been configured for `docker` should produce
the same output as above:

```bash
$ docker run --runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all ubuntu:20.04 nvidia-smi -L
GPU 0: NVIDIA A100-SXM4-40GB (UUID: GPU-4cf8db2d-06c0-7d70-1a51-e59b25b2c16c)
GPU 1: NVIDIA A100-SXM4-40GB (UUID: GPU-4404041a-04cf-1ccf-9e70-f139a9b1e23c)
GPU 2: NVIDIA A100-SXM4-40GB (UUID: GPU-79a2ba02-a537-ccbf-2965-8e9d90c0bd54)
GPU 3: NVIDIA A100-SXM4-40GB (UUID: GPU-662077db-fa3f-0d8f-9502-21ab0ef058a2)
GPU 4: NVIDIA A100-SXM4-40GB (UUID: GPU-ec9d53cc-125d-d4a3-9687-304df8eb4749)
GPU 5: NVIDIA A100-SXM4-40GB (UUID: GPU-3eb87630-93d5-b2b6-b8ff-9b359caf4ee2)
GPU 6: NVIDIA A100-SXM4-40GB (UUID: GPU-8216274a-c05d-def0-af18-c74647300267)
GPU 7: NVIDIA A100-SXM4-40GB (UUID: GPU-b1028956-cfa2-0990-bf4a-5da9abb51763)
```

If you have the `nvidia-container-toolkit` installed, but you have an error
when trying to run the `docker` command above, skip to the [Setup](#setup)
section below to see if some of the configuration steps there resolve the
issue.

## Setup

With all of the prerequisites installed, run the following commands to
configure the `nvidia-container-toolkit` for use with `kind`.

```bash
sudo nvidia-ctk runtime configure --runtime=docker --set-as-default --cdi.enabled
sudo nvidia-ctk config --set accept-nvidia-visible-devices-as-volume-mounts=true --in-place
sudo systemctl restart docker
```

The first command ensures that `docker` is configured for use with the toolkit
and that the `nvidia` runtime is set as its default. The second command enables
a feature flag of the toolkit as described in this [this
document](https://docs.google.com/document/d/1uXVF-NWZQXgP1MLb87_kMkQvidpnkNWicdpO2l9g-fw/edit#)).
This feature is leveraged to allow us to inject GPU support into each `kind`
worker node.

To ensure that this feature has been enabled correctly, run the following and
verify you get output similar to the following:

```bash
$ docker run -v /dev/null:/var/run/nvidia-container-devices/all ubuntu:20.04 nvidia-smi -L
GPU 0: NVIDIA A100-SXM4-40GB (UUID: GPU-4cf8db2d-06c0-7d70-1a51-e59b25b2c16c)
GPU 1: NVIDIA A100-SXM4-40GB (UUID: GPU-4404041a-04cf-1ccf-9e70-f139a9b1e23c)
GPU 2: NVIDIA A100-SXM4-40GB (UUID: GPU-79a2ba02-a537-ccbf-2965-8e9d90c0bd54)
GPU 3: NVIDIA A100-SXM4-40GB (UUID: GPU-662077db-fa3f-0d8f-9502-21ab0ef058a2)
GPU 4: NVIDIA A100-SXM4-40GB (UUID: GPU-ec9d53cc-125d-d4a3-9687-304df8eb4749)
GPU 5: NVIDIA A100-SXM4-40GB (UUID: GPU-3eb87630-93d5-b2b6-b8ff-9b359caf4ee2)
GPU 6: NVIDIA A100-SXM4-40GB (UUID: GPU-8216274a-c05d-def0-af18-c74647300267)
GPU 7: NVIDIA A100-SXM4-40GB (UUID: GPU-b1028956-cfa2-0990-bf4a-5da9abb51763)
```

## Create a cluster

This repository contains two scripts that can be used to create `kind` clusters
with GPU support in different ways:

```bash
./evenly-distributed-gpus-by-workers.sh <num-workers>
./explicit-gpus-for-workers.sh <worker1-gpu-list> <worker2-gpu-list> ...
```

The first can be used to create a cluster with a specific number of workers,
where the number of GPUs on the machine are evenly distributed among them. The
second can be used to customize exactly how many workers there are and which
exact GPUs are placed on them.

For example, on an 8 GPU machine, running the following will create a `kind`
cluster with 4 worker nodes that have access to 2 GPUs each:

```bash
./evenly-distributed-gpus-by-workers.sh 4
```

Likewise, running the following will create a `kind` cluster with 8 worker
nodes that have access to 1 GPU each:

```bash
./evenly-distributed-gpus-by-workers.sh 8
```

And running the following will create a `kind` cluster with 2 worker nodes that
have access to the specific GPUs that are listed on the command line for each
worker node (the first with access to GPU 0 and the second with access to GPUs
1, 2, and 3):

```bash
./explicit-gpus-for-workers.sh "0" "1 2 3"
```

This can be verified by running the following:
```bash
$ kind get clusters
evenly-distributed-2-by-4
evenly-distributed-1-by-8
explicit-gpus-xrq0c
```

```bash
$ (source common.sh; print_all_worker_gpus evenly-distributed-2-by-4)
...
```

```bash
$ (source common.sh; print_all_worker_gpus evenly-distributed-1-by-8)
...
```

```bash
$ (source common.sh; print_all_worker_gpus explicit-gpus-xrq0c)
...
```

Where the output of the first looks like the following (with similar results for
the others):
```bash
{
  "node": "evenly-distributed-2-by-4-worker",
  "nvidia.com/gpu": [
    {
      "index": "0",
      "name": "NVIDIA A100-SXM4-40GB",
      "uuid": "GPU-4cf8db2d-06c0-7d70-1a51-e59b25b2c16c"
    },
    {
      "index": "1",
      "name": "NVIDIA A100-SXM4-40GB",
      "uuid": "GPU-4404041a-04cf-1ccf-9e70-f139a9b1e23c"
    }
  ]
}
{
  "node": "evenly-distributed-2-by-4-worker2",
  "nvidia.com/gpu": [
    {
      "index": "0",
      "name": "NVIDIA A100-SXM4-40GB",
      "uuid": "GPU-79a2ba02-a537-ccbf-2965-8e9d90c0bd54"
    },
    {
      "index": "1",
      "name": "NVIDIA A100-SXM4-40GB",
      "uuid": "GPU-662077db-fa3f-0d8f-9502-21ab0ef058a2"
    }
  ]
}
{
  "node": "evenly-distributed-2-by-4-worker3",
  "nvidia.com/gpu": [
    {
      "index": "0",
      "name": "NVIDIA A100-SXM4-40GB",
      "uuid": "GPU-ec9d53cc-125d-d4a3-9687-304df8eb4749"
    },
    {
      "index": "1",
      "name": "NVIDIA A100-SXM4-40GB",
      "uuid": "GPU-3eb87630-93d5-b2b6-b8ff-9b359caf4ee2"
    }
  ]
}
{
  "node": "evenly-distributed-2-by-4-worker4",
  "nvidia.com/gpu": [
    {
      "index": "0",
      "name": "NVIDIA A100-SXM4-40GB",
      "uuid": "GPU-8216274a-c05d-def0-af18-c74647300267"
    },
    {
      "index": "1",
      "name": "NVIDIA A100-SXM4-40GB",
      "uuid": "GPU-b1028956-cfa2-0990-bf4a-5da9abb51763"
    }
  ]
}
```

## Install the k8s-device-plugin

With one of the clusters above in place, the `k8s-device-plugin` (or
`gpu-operator`) can be installed on the cluster as appropriate. For the
purposes of this example, we will install the `k8s-device-plugin` directly.

First, add the `helm`repo for the `k8s-device-plugin`:
```bash
helm repo add nvdp https://nvidia.github.io/k8s-device-plugin
helm repo update
```

Then pick the cluster you want to install to:
```
export KIND_CLUSTER_NAME=evenly-distributed-2-by-4
```

And install the `k8s-device-plugin` as follows:
```bash
helm upgrade -i \
    --kube-context=kind-${KIND_CLUSTER_NAME} \
    --namespace nvidia \
    --create-namespace \
    nvidia-device-plugin nvdp/nvidia-device-plugin
```

Running the following we can see the pods for the plugin coming online:
```bash
$ kubectl --context=kind-${KIND_CLUSTER_NAME} get pod -n nvidia
NAME                         READY   STATUS    RESTARTS   AGE
nvidia-device-plugin-9lfxq   1/1     Running   0          15s
nvidia-device-plugin-hxvzb   1/1     Running   0          15s
nvidia-device-plugin-lgt85   1/1     Running   0          15s
nvidia-device-plugin-r5zbm   1/1     Running   0          15s
```

Running the following verifies we have 4 nodes with 2 allocatable GPUs each:
```bash
$ kubectl --context=kind-${KIND_CLUSTER_NAME} get nodes -o json | jq -r '.items[] | select(.metadata.name | test("-worker[0-9]*$")) | {name: .metadata.name, "nvidia.com/gpu": .status.allocatable["nvidia.com/gpu"]}'
{
  "name": "evenly-distributed-2-by-4-worker",
  "nvidia.com/gpu": "2"
}
{
  "name": "evenly-distributed-2-by-4-worker2",
  "nvidia.com/gpu": "2"
}
{
  "name": "evenly-distributed-2-by-4-worker3",
  "nvidia.com/gpu": "2"
}
{
  "name": "evenly-distributed-2-by-4-worker4",
  "nvidia.com/gpu": "2"
}
```

Running the following verifies that a workload can be deployed and run on a set
of GPUs in this cluster:

```bash
cat << EOF | kubectl --context=kind-${KIND_CLUSTER_NAME} apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: gpu-test
spec:
  restartPolicy: OnFailure
  containers:
  - name: ctr
    image: ubuntu:22.04
    command: ["nvidia-smi", "-L"]
    resources:
      limits:
        nvidia.com/gpu: 2
EOF
```

```bash
$ kubectl --context=kind-${KIND_CLUSTER_NAME} logs gpu-test
GPU 0: NVIDIA A100-SXM4-40GB (UUID: GPU-4cf8db2d-06c0-7d70-1a51-e59b25b2c16c)
GPU 1: NVIDIA A100-SXM4-40GB (UUID: GPU-4404041a-04cf-1ccf-9e70-f139a9b1e23c)
```

## Delete all clusters

The following command can be used to delete all `kind` clusters:

```bash
for cluster in $(kind get clusters); do kind delete cluster --name=${cluster}; done
```
