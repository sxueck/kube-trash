package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/sxueck/kube-trash/config"
	"github.com/sxueck/kube-trash/internal"
	"github.com/sxueck/kube-trash/internal/cluster"
	"github.com/sxueck/kube-trash/pkg/storage"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/util/workqueue"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, unix.SIGQUIT, unix.SIGTERM, unix.SIGINT)

	if err := Run(ctx); err != nil {
		log.Printf("error running program: %v", err)
		cancel()
	}

	select {
	case <-sigterm:
		log.Println("stop and clean all processes")
		cancel()
	case <-ctx.Done():
	}

	<-time.NewTicker(1 * time.Second).C
}

func Run(ctx context.Context) error {
	errChan := make(chan error, 1)
	restConfig, err := cluster.NewClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientSet, err := cluster.NewClusterClient(restConfig)
	if err != nil {
		return err
	}
	dynamicClient, err := cluster.NewClusterDynamicClient(restConfig)
	if err != nil {
		return err
	}
	discoveryClient, err := cluster.NewClusterDiscoveryClient(restConfig)
	if err != nil {
		return err
	}

	asyncQueue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	storageConfig := config.GlobalCfg.Storage
	s3, err := storage.NewS3Storage(storageConfig)
	if err != nil {
		return err
	}

	// Start Resource Monitoring
	go func(errChan chan<- error) {
		err = internal.ServResourcesInformer(cluster.ClientSet{
			BaseClient:      clientSet,
			DiscoveryClient: discoveryClient,
			DynamicClient:   dynamicClient,
		}, asyncQueue)
		if err != nil {
			errChan <- fmt.Errorf("error in ServResourcesInformer: %v", err)
		}
	}(errChan)

	// Start polling for items in the queue
	// Note: that go routines are independent of the stack, so they are not affected by the function lifecycle
	go processQueue(ctx, asyncQueue, s3)

	select {
	case err = <-errChan:
		return err
	case <-ctx.Done():
		return nil
	}
}

func processQueue(ctx context.Context, q workqueue.RateLimitingInterface, s3 *storage.S3Storage) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down the queue due to context cancellation")
			return
		default:
			element, shutdown := q.Get()
			if shutdown {
				log.Println("Shutting down the queue")
				return
			}

			item, ok := element.(*config.QueueItem)
			if !ok {
				log.Printf("Invalid element type, expected config.QueueItem, got %T", element)
				q.Done(element)
				continue
			}
			// Use s3 to store files
			log.Printf("Processing item: %+v", item.Name)
			objectName := GenMinioCompleteObjectName(item.Namespace, item.Name, item.Kind)
			err := s3.Upload(ctx,
				objectName,
				bytes.NewReader(item.Data),
				int64(len(item.Data)))
			if err != nil {
				log.Printf("Error uploading to storage: %v", err)
			} else {
				log.Printf("Successfully uploaded %s to storage", objectName)
			}
		}
	}
}

func GenMinioCompleteObjectName(namespace, name, kind string) string {
	return fmt.Sprintf("%s/%s_%s_%s.yaml", namespace, name, kind, time.Now().Format("2006-01-02_150405"))
}
