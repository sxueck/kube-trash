package cluster

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/sxueck/kube-trash/config"
	"k8s.io/client-go/discovery"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientSet struct {
	BaseClient      kubernetes.Interface
	DynamicClient   *dynamic.DynamicClient
	DiscoveryClient *discovery.DiscoveryClient
}

// NewClientConfig creates and returns a new Kubernetes client configuration.
//
// This function attempts to create a client configuration in the following order:
// 1. If both APIServer and KubeConfig are provided, it uses these to build the configuration.
// 2. If only KubeConfig is provided, it extracts the API server from the kubeconfig file.
// 3. If neither APIServer nor KubeConfig are provided, it attempts to use the in-cluster configuration.
//
// The function uses GlobalCfg.APIServer and GlobalCfg.KubeConfig for configuration.
//
// Returns:
//   - *rest.Config: A pointer to the REST client configuration if successful.
//   - error: An error if the configuration could not be created.
func NewClientConfig() (*rest.Config, error) {
	var (
		restConfig *rest.Config
		err        error
	)

	kubeConfigPath := config.GlobalCfg.KubeConfig
	var apiServer string

	if len(kubeConfigPath) > 0 {
		// extract the apiServer address from the kubeConfig file
		cfg, err := clientcmd.LoadFromFile(kubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
		}

		if len(cfg.Clusters) == 0 {
			return nil, fmt.Errorf("no clusters defined in kubeconfig")
		}

		for _, cluster := range cfg.Clusters {
			apiServer = cluster.Server
			break
		}

		fmt.Printf("Extracted API server from kubeconfig: %s\n", apiServer)
	}

	if len(apiServer) > 0 {
		fmt.Printf("Using API server: %s\n", apiServer)
		if len(kubeConfigPath) == 0 {
			return nil, fmt.Errorf("KubeConfig is not set")
		}

		restConfig, err = clientcmd.BuildConfigFromFlags(apiServer, kubeConfigPath)
		if err != nil {
			return nil, err
		}
	} else {
		log.Infoln("Using in-cluster configuration")
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Errorf("Failed to get in-cluster config: %v", err)
			return nil, err
		}
	}

	return restConfig, nil
}

func NewClusterClient(restConfig *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(restConfig)
}

func NewClusterDynamicClient(restConfig *rest.Config) (*dynamic.DynamicClient, error) {
	return dynamic.NewForConfig(restConfig)
}

func NewClusterDiscoveryClient(restConfig *rest.Config) (*discovery.DiscoveryClient, error) {
	return discovery.NewDiscoveryClientForConfig(restConfig)
}
