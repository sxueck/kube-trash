package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/sxueck/kube-trash/config"
	"github.com/sxueck/kube-trash/internal/cluster"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
)

const (
	informerReSyncPeriod   = time.Minute
	ListenerMaxConcurrency = 20 // The maximum number of concurrent listeners
)

// ServResourcesInformer Reduce the impact on API servers by controlling the number of concurrent queries
func ServResourcesInformer(ctx context.Context, client cluster.ClientSet, asyncQueue workqueue.RateLimitingInterface) error {
	// Gets a list of resources supported by the server
	resourceList, err := client.BaseClient.Discovery().ServerPreferredResources()
	if err != nil {
		return fmt.Errorf("failed to get server preferred resources: %w", err)
	}

	// Concurrency is controlled using semaphore
	semaphore := make(chan struct{}, ListenerMaxConcurrency)
	var wg sync.WaitGroup

	// Iterate over all resources, creating informer for each resource
	for _, resource := range resourceList {
		gv, err := schema.ParseGroupVersion(resource.GroupVersion)
		if err != nil {
			log.Warnf("failed to parse group version: %s", err)
			continue
		}

		for _, apiResource := range resource.APIResources {
			if filterSkipResource(apiResource) {
				continue
			}

			gvr, err := gvkToGVR(client.DiscoveryClient, schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    apiResource.Kind,
			})
			if err != nil {
				log.Warnf("Error converting GVK to GVR: %v", err)
				continue
			}

			wg.Add(1)
			go func(gvr schema.GroupVersionResource) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire semaphore
				defer func() { <-semaphore }() // Release semaphore

				if queueErr := runInformer(ctx, client.DynamicClient, gvr, asyncQueue); queueErr != nil {
					if !errors.IsNotFound(queueErr) {
						log.Warnf("Error running informer for %s: %v", gvr.String(), queueErr)
					}
				}
			}(gvr)
		}
	}

	wg.Wait()
	return nil
}

// runInformer Runs informer for the specified resource
// Here we need to pass in a queue for asynchronous processing, to avoid the process of blocking the cluster due to i/o problems
func runInformer(ctx context.Context, dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, asyncQueue workqueue.RateLimitingInterface) error {
	var informer cache.SharedIndexInformer
	err := retry.OnError(retry.DefaultRetry, func(err error) bool {
		_, isNotFound := err.(*errors.StatusError)
		return isNotFound
	}, func() error {
		informer = createDynamicInformer(dynamicClient, gvr)
		_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				u := ExtractAboutKeyInformation(obj)
				log.Infof("Record Delete Events -  %s %s %s\n", u.Namespace, u.Kind, u.Name)
				asyncQueue.Add(u)
			},
		})
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to create informer for %s after retries: %w", gvr.String(), err)
	}

	go informer.Run(ctx.Done())

	// Add a timeout for cache synchronization
	syncCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if !cache.WaitForCacheSync(syncCtx.Done(), informer.HasSynced) {
		log.Warnf("Failed to sync cache for %s, continuing without full synchronization", gvr.String())
		log.Warnf("Detailed cache sync failure for %s: %v", gvr.String(), syncCtx.Err())
	}

	//<-ctx.Done()
	//log.Infof("Context done, stopping informer for %s", gvr.String())
	return nil
}

// gvkToGVR Convert GroupVersionKind to GroupVersionResource
func gvkToGVR(discoveryClient *discovery.DiscoveryClient, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	apiResources, err := discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("error getting server resources for group version %s: %w", gvk.GroupVersion().String(), err)
	}

	for _, apiResource := range apiResources.APIResources {
		if apiResource.Kind == gvk.Kind {
			return schema.GroupVersionResource{
				Group:    gvk.Group,
				Version:  gvk.Version,
				Resource: apiResource.Name,
			}, nil
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("no resource found for GVK %s", gvk)
}

func createDynamicInformer(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return dynamicClient.Resource(gvr).Namespace(v1.NamespaceAll).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return dynamicClient.Resource(gvr).Namespace(v1.NamespaceAll).Watch(context.TODO(), options)
			},
		},
		&unstructured.Unstructured{},
		informerReSyncPeriod,
		cache.Indexers{},
	)
}

func filterSkipResource(apiResource v1.APIResource) bool {
	excludeResources := config.GlobalCfg.Watch.ExcludeResource
	includeResources := config.GlobalCfg.Watch.IncludeResource

	// highest priority decision
	for _, resource := range excludeResources {
		if apiResource.Name == resource {
			return true
		}
	}

	// secondary priority decision
	for _, resource := range includeResources {
		if apiResource.Name == resource {
			return false
		}
	}

	// If includeResources contains "*", do not skip any resources
	for _, resource := range includeResources {
		if resource == "*" {
			return false
		}
	}

	// if is not specified it is skipped by default
	return true
}

func interfaceToYAML(obj interface{}) ([]byte, error) {
	yamlData, err := yaml.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object to YAML: %w", err)
	}

	// Here to add a layer of processing logic, because unstructured.
	// Unstructured is the root of "object" key, input here general YAML formats reflection to remove the root keys
	/* like
	object:
		apiVersion: v1
		kind: Pod
		metadata:
		creationTimestamp: "2024-09-13T06:44:02Z"
		deletionGracePeriodSeconds: 0
	*/
	var rawData map[string]interface{}
	if err = yaml.Unmarshal(yamlData, &rawData); err != nil {
		log.Warnf("Error unmarshalling data: %v", err)
		return yamlData, nil
	}

	if obj, exists := rawData["object"]; exists {
		objData, err := yaml.Marshal(obj)
		if err != nil {
			log.Warnf("Error marshalling object data: %v", err)
			return yamlData, nil
		}
		return objData, nil
	}

	log.Infoln("'object' field does not exist in the data")
	return yamlData, nil
}

func ExtractAboutKeyInformation(obj interface{}) *config.QueueItem {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Warnf("Invalid object type, expected *unstructured.Unstructured, got %T", obj)
		return nil
	}

	yamlContent, err := interfaceToYAML(obj)
	if err != nil {
		log.Warnf("Error converting object to YAML: %v", err)
		return nil
	}

	return &config.QueueItem{
		Name:      unstructuredObj.GetName(),
		Namespace: unstructuredObj.GetNamespace(),
		Kind:      unstructuredObj.GetKind(),
		Data:      yamlContent,
	}
}
