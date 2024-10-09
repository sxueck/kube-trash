package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/sxueck/kube-trash/config"
	"github.com/sxueck/kube-trash/internal"
	"github.com/sxueck/kube-trash/internal/cluster"
	"github.com/sxueck/kube-trash/pkg/storage"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/util/workqueue"

	log "github.com/sirupsen/logrus"
)

func init() {
	config.InitConfig()
	config.InitLogComponent()
	log.Debugf("%+v\n", config.GlobalCfg)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, unix.SIGQUIT, unix.SIGTERM, unix.SIGINT)

	go func() {
		if err := Run(ctx); err != nil {
			log.Warnf("error running program: %v", err)
			cancel()
		}
	}()

	select {
	case <-sigterm:
		log.Infoln("Received termination signal. Stopping and cleaning all processes...")
		cancel()
	case <-ctx.Done():
		log.Infoln("Context canceled. Exiting...")
	}

	time.Sleep(3 * time.Second)
	log.Infoln("Program exited.")
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
		err = internal.ServResourcesInformer(ctx, cluster.ClientSet{
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
			log.Infoln("Shutting down the queue due to context cancellation")
			return
		default:
			element, shutdown := q.Get()
			if shutdown {
				log.Infoln("Shutting down the queue")
				return
			}

			item, ok := element.(*config.QueueItem)
			if !ok {
				log.Infof("Invalid element type, expected config.QueueItem, got %T", element)
				q.Done(element)
				continue
			}
			// Use s3 to store files
			log.Infof("Processing item: %+v", item.Name)
			objectName := GenMinioCompleteObjectName(item.Namespace, item.Name, item.Kind)
			err := s3.Upload(ctx,
				objectName,
				bytes.NewReader(item.Data),
				int64(len(item.Data)))
			if err != nil {
				log.Errorf("Error uploading to storage: %v", err)
			} else {
				log.Infof("Successfully uploaded %s to storage", objectName)
			}
		}
	}
}

func GenMinioCompleteObjectName(namespace, name, kind string) string {
	return fmt.Sprintf("%s/%s_%s_%s.yaml", namespace, name, kind, time.Now().Format("2006-01-02_150405"))
}
