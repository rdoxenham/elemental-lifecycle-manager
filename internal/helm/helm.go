package helm

import (
	"fmt"

	releasev1 "helm.sh/helm/v4/pkg/release/v1"
	helmstorage "helm.sh/helm/v4/pkg/storage"
	helmdriver "helm.sh/helm/v4/pkg/storage/driver"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// ChartState represents the current state of a Helm chart upgrade.
type ChartState int

const (
	ChartStateUnknown ChartState = iota
	ChartStateNotInstalled
	ChartStateVersionAlreadyInstalled
	ChartStateInProgress
	ChartStateFailed
	ChartStateSucceeded
)

// String returns a human-readable representation of the chart state.
func (s ChartState) String() string {
	switch s {
	case ChartStateNotInstalled:
		return "not installed"
	case ChartStateVersionAlreadyInstalled:
		return "version already installed"
	case ChartStateInProgress:
		return "in progress"
	case ChartStateFailed:
		return "failed"
	case ChartStateSucceeded:
		return "succeeded"
	default:
		return "unknown"
	}
}

var ErrReleaseNotFound = helmdriver.ErrReleaseNotFound

// ReleaseInfo contains relevant information about a Helm release
// needed for upgrade operations.
type ReleaseInfo struct {
	ChartVersion string
	Namespace    string
	Config       map[string]any
}

// Client provides access to Helm release information.
type Client interface {
	RetrieveRelease(name string) (*ReleaseInfo, error)
}

// StorageClient implements Client using Helm storage.
type StorageClient struct {
	*helmstorage.Storage
}

// NewClient creates a Helm storage client using the in-cluster config.
func NewClient() (*StorageClient, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("retrieving cluster config: %w", err)
	}

	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	driver := helmdriver.NewSecrets(k8sClient.CoreV1().Secrets(""))
	storage := helmstorage.Init(driver)

	return &StorageClient{storage}, nil
}

// RetrieveRelease retrieves the latest Helm release by name.
func (c *StorageClient) RetrieveRelease(name string) (*ReleaseInfo, error) {
	rel, err := c.Storage.Last(name)
	if err != nil {
		return nil, err
	}

	release, ok := rel.(*releasev1.Release)
	if !ok {
		return nil, fmt.Errorf("unexpected release type: %T", rel)
	}

	return &ReleaseInfo{
		ChartVersion: release.Chart.Metadata.Version,
		Namespace:    release.Namespace,
		Config:       release.Config,
	}, nil
}
