package nvkind

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/NVIDIA/go-nvlib/pkg/nvml"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/rand"
	kind "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
)

func NewConfig(opts ...ConfigOption) (*Config, error) {
	o := ConfigOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.defaultName == "" {
		o.defaultName = fmt.Sprintf("nvkind-%s", rand.String(5))
	}
	if o.nvml == nil {
		o.nvml = nvml.New()
	}
	if o.stdout == nil {
		o.stdout = os.Stdout
	}
	if o.stderr == nil {
		o.stderr = os.Stderr
	}
	if o.configTemplate == nil && o.configTemplatePath == "" {
		o.configTemplate = defaultConfigTemplate
	}
	if o.configValues == nil && o.configValuesPath == "" {
		o.configValues = defaultConfigValues
	}
	if o.configTemplate == nil && o.configTemplatePath != "" {
		data, err := os.ReadFile(o.configTemplatePath)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}
		o.configTemplate = data
	}
	if o.configValues == nil && o.configValuesPath != "" {
		data, err := os.ReadFile(o.configValuesPath)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}
		o.configValues = data
	}

	tmpl, err := template.New("configTemplate").Funcs(o.buildFuncMap()).Parse(string(o.configTemplate))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var values any
	if err := yaml.Unmarshal(o.configValues, &values); err != nil {
		return nil, fmt.Errorf("unmarshaling YAML: %w", err)
	}
	values = convertToMap(values)

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, values); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	var cluster kind.Cluster
	if err := yaml.Unmarshal(buffer.Bytes(), &cluster); err != nil {
		return nil, fmt.Errorf("unmarshaling YAML: %w", err)
	}

	if cluster.Name == "" {
		cluster.Name = o.defaultName
	}

	if o.image != "" {
		for i := range cluster.Nodes {
			cluster.Nodes[i].Image = o.image
		}
	}

	config := &Config{
		Cluster: &cluster,
		nvml:    o.nvml,
		stdout:  o.stdout,
		stderr:  o.stderr,
	}

	return config, nil
}

func (o *ConfigOptions) buildFuncMap() template.FuncMap {
	funcmap := map[string]any{
		"numGPUs": o.numGPUs,
	}
	for k, v := range o.extraFuncMap {
		funcmap[k] = v
	}
	for k, v := range sprig.FuncMap() {
		funcmap[k] = v
	}
	return funcmap
}

func (o *ConfigOptions) numGPUs() (int, error) {
	if ret := o.nvml.Init(); ret != nvml.SUCCESS {
		return -1, fmt.Errorf("running nvml.Init: %w", ret)
	}
	defer func() { _ = o.nvml.Shutdown() }()

	numGPUs, ret := o.nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return -1, fmt.Errorf("running nvml.DeviceGetCount: %w", ret)
	}

	return numGPUs, nil
}

func convertToMap(data any) any {
	switch v := data.(type) {
	case map[any]any:
		result := make(map[string]any)
		for key, val := range v {
			//nolint:forcetypeassert
			result[key.(string)] = convertToMap(val)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = convertToMap(item)
		}
		return result
	default:
		return v
	}
}
