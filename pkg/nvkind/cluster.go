package nvkind

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	kind "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
)

const (
	nvkindClusterConfigName = "nvkind-cluster-config"
)

func GetClusterNames() (sets.Set[string], error) {
	command := []string{
		"kind", "get", "clusters", "-q",
	}

	cmd := exec.Command(command[0], command[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("executing command: %w", err)
	}

	return sets.New(strings.Fields(string(output))...), nil
}

func NewCluster(opts ...ClusterOption) (*Cluster, error) {
	o := ClusterOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	if err := o.setConfig(); err != nil {
		return nil, fmt.Errorf("setting config: %w", err)
	}
	if o.name == "" {
		o.name = o.config.Name
	}
	if o.kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			o.kubeconfig = home + "/.kube/config"
		}
	}

	cluster := &Cluster{
		Name:       o.name,
		config:     o.config.Cluster,
		kubeconfig: o.kubeconfig,
		nvml:       o.config.nvml,
		stdout:     o.config.stdout,
		stderr:     o.config.stderr,
	}

	return cluster, nil
}

func (c *Cluster) Create(opts ...ClusterCreateOption) error {
	command := []string{
		"kind", "create", "cluster",
		"--name", c.Name,
		"--config", "-",
	}

	o := ClusterCreateOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.retain {
		command = append(command, "--retain")
	}
	if o.wait != 0 {
		command = append(command, "--wait", o.wait.String())
	}

	configBytes, err := yaml.Marshal(c.config)
	if err != nil {
		return fmt.Errorf("marshaling YAML: %w", err)
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = bytes.NewBuffer(configBytes)
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("executing command: %w", err)
	}

	if err := addConfigBytesToExistingCluster(c.Name, configBytes); err != nil {
		return fmt.Errorf("adding config to cluster: %w", err)
	}

	return nil
}

func (c *Cluster) Delete() error {
	command := []string{
		"kind", "delete", "cluster",
		"--name", c.Name,
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("executing command: %w", err)
	}

	return nil
}

func (c *Cluster) GetNodes() ([]Node, error) {
	command := []string{
		"kind", "get", "nodes",
		"--name", c.Name,
	}

	cmd := exec.Command(command[0], command[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("executing command: %w", err)
	}

	nodeNamesList := strings.Fields(string(output))
	sort.Strings(nodeNamesList)

	nodeNames := make(map[kind.NodeRole][]string)
	for _, node := range nodeNamesList {
		trimmed := strings.TrimRightFunc(node, unicode.IsDigit)
		if strings.HasSuffix(trimmed, string(kind.ControlPlaneRole)) {
			nodeNames[kind.ControlPlaneRole] = append(nodeNames[kind.ControlPlaneRole], node)
			continue
		}
		if strings.HasSuffix(trimmed, string(kind.WorkerRole)) {
			nodeNames[kind.WorkerRole] = append(nodeNames[kind.WorkerRole], node)
			continue
		}
		return nil, fmt.Errorf("unable to determine node role from name: %v", node)
	}

	nodeConfigs := make(map[kind.NodeRole][]*kind.Node)
	for _, node := range c.config.Nodes {
		if node.Role == kind.ControlPlaneRole {
			nodeConfigs[node.Role] = append(nodeConfigs[node.Role], &node)
			continue
		}
		if node.Role == kind.WorkerRole {
			nodeConfigs[node.Role] = append(nodeConfigs[node.Role], &node)
			continue
		}
		return nil, fmt.Errorf("unknown node role: %v", node.Role)
	}

	nodes := make([]Node, 0, len(nodeNamesList))
	for _, role := range []kind.NodeRole{kind.ControlPlaneRole, kind.WorkerRole} {
		if len(nodeNames[role]) != len(nodeConfigs[role]) {
			return nil, fmt.Errorf("node names and configs mismatch for %v role", role)
		}

		for i := range nodeNames[role] {
			node := Node{
				Name:   nodeNames[role][i],
				config: nodeConfigs[role][i].DeepCopy(),
				nvml:   c.nvml,
				stdout: c.stdout,
				stderr: c.stderr,
			}
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

func (o *ClusterOptions) setConfig() error {
	existingClusters, err := GetClusterNames()
	if err != nil {
		return fmt.Errorf("getting list of existing clusters: %w", err)
	}

	if o.name != "" && o.config != nil {
		o.config.Cluster.Name = o.name
	}

	if !existingClusters.Has(o.name) && o.config != nil {
		return nil
	}

	var options []ConfigOption
	if existingClusters.Has(o.name) {
		existingConfigBytes, err := getConfigBytesFromExistingCluster(o.name)
		if err != nil {
			return fmt.Errorf("getting config bytes: %w", err)
		}
		if o.config != nil {
			var existingConfig kind.Cluster
			if err := yaml.Unmarshal(existingConfigBytes, &existingConfig); err != nil {
				return fmt.Errorf("unmarshaling YAML: %w", err)
			}
			if !reflect.DeepEqual(&existingConfig, o.config.Cluster) {
				return fmt.Errorf("cannot pass new config to existing cluster")
			}
			return nil
		}
		options = append(options, WithConfigTemplate(existingConfigBytes))
	}

	config, err := NewConfig(options...)
	if err != nil {
		return fmt.Errorf("creating new config: %w", err)
	}
	o.config = config

	return nil
}

func addConfigBytesToExistingCluster(name string, configBytes []byte) error {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: "kind-" + name}
	loadingConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, configOverrides)
	csconfig, err := loadingConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("loading client config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(csconfig)
	if err != nil {
		return fmt.Errorf("creating clientset: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: nvkindClusterConfigName,
		},
		Data: map[string]string{
			"config": string(configBytes),
		},
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := clientset.CoreV1().ConfigMaps("default").Create(context.Background(), configMap, metav1.CreateOptions{})
		return err
	})
	if retryErr != nil {
		return fmt.Errorf("writing configmap: %w", err)
	}

	return nil
}

func getConfigBytesFromExistingCluster(name string) ([]byte, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: "kind-" + name}
	loadingConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, configOverrides)
	csconfig, err := loadingConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("loading client config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(csconfig)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %w", err)
	}

	configMap, err := clientset.CoreV1().ConfigMaps("default").Get(context.Background(), nvkindClusterConfigName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting configmap: %w", err)
	}

	return []byte(configMap.Data["config"]), nil
}
