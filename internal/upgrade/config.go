/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package upgrade

import (
	"fmt"
	"strings"

	"github.com/suse/elemental/v3/pkg/manifest/api"
	"github.com/suse/elemental/v3/pkg/manifest/resolver"
)

// NewConfig creates a release upgrade specification from the resolved manifest.
// The upgrade is built by extracting configuration from the core platform
// and optionally merging with product extension components.
func NewConfig(manifest *resolver.ResolvedManifest, releaseName string) (*Config, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}

	if manifest.CorePlatform == nil {
		return nil, fmt.Errorf("core platform manifest is required")
	}

	core := manifest.CorePlatform
	config := &Config{
		ReleaseName: releaseName,
		Version:     core.Metadata.Version,
		OS: &SUCPlanConfig{
			Image:   core.Components.OperatingSystem.Image.Base,
			Version: core.Metadata.Version,
		},
	}

	kubernetesImage, kubernetesVersion := extractKubernetesImage(&core.Components.Systemd)
	if kubernetesImage == "" {
		return nil, fmt.Errorf("kubernetes image is required but not found in release manifest")
	}

	config.Kubernetes = &SUCPlanConfig{
		Image:   kubernetesImage,
		Version: kubernetesVersion,
	}

	if manifest.ProductExtension == nil {
		config.HelmCharts = helmChartConfig(releaseName, config.Version, core.Components.Helm, nil)
	} else {
		product := manifest.ProductExtension
		config.HelmCharts = helmChartConfig(releaseName, config.Version, core.Components.Helm, product.Components.Helm)
	}

	return config, nil
}

// helmChartConfig merges Helm configurations from core and product manifests.
func helmChartConfig(releaseName, version string, core *api.Helm, product *api.Helm) *HelmChartConfig {
	config := &HelmChartConfig{
		ReleaseName:    releaseName,
		ReleaseVersion: version,
		Charts:         make([]*api.HelmChart, 0),
		Repositories:   make([]*api.HelmRepository, 0),
	}

	// Add core charts and repositories
	if core != nil {
		config.Charts = append(config.Charts, core.Charts...)
		config.Repositories = append(config.Repositories, core.Repositories...)
	}

	// Add product charts and repositories
	if product != nil {
		config.Charts = append(config.Charts, product.Charts...)
		config.Repositories = append(config.Repositories, product.Repositories...)
	}

	if len(config.Charts) == 0 && len(config.Repositories) == 0 {
		return nil
	}

	return config
}

func extractKubernetesImage(systemd *api.Systemd) (image string, version string) {
	if systemd == nil {
		return "", ""
	}

	// Extract version from the respective systemd extension
	for _, ext := range systemd.Extensions {
		if strings.Contains(ext.Name, "rke2") {
			image = ext.Image
			break
		}
	}

	res := strings.Split(image, ":")
	if len(res) != 2 {
		return "", ""
	}

	return res[0], res[1]
}
