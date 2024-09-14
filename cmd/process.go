package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sxueck/kube-trash/config"
	"github.com/sxueck/kube-trash/internal"
	"github.com/sxueck/kube-trash/internal/cluster"
	"github.com/sxueck/kube-trash/pkg/queue"
	"github.com/sxueck/kube-trash/pkg/storage"
)

func Run() error {
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

	asyncQueue := queue.NewQueue()

	minioConfig := config.GlobalCfg.Storage
	minioStorage, err := storage.NewMinioStorage(minioConfig)
	if err != nil {
		return err
	}

	// Start Resource Monitoring
	go func() {
		err := internal.ServResourcesInformer(cluster.ClientSet{
			BaseClient:      clientSet,
			DiscoveryClient: discoveryClient,
			DynamicClient:   dynamicClient,
		}, asyncQueue)
		if err != nil {
			log.Printf("Error in ServResourcesInformer: %v", err)
		}
	}()

	// Start polling for items in the queue
	go processQueue(asyncQueue, minioStorage)

	// The main thread stays running
	select {}
}

func processQueue(q *queue.Queue, minio *storage.MinioStorage) {
	for {
		item, ok := q.Dequeue()
		if !ok {
			time.Sleep(time.Second * 40)
			continue
		}

		// Use MinIO to store files
		ctx := context.Background()
		log.Printf("Processing item: %+v", item.Name)
		objectName := GenMinioCompleteObjectName(item.Namespace, item.Name, item.Kind)
		err := minio.Upload(ctx,
			objectName,
			bytes.NewReader(item.Data),
			int64(len(item.Data)))
		if err != nil {
			log.Printf("Error uploading to MinIO: %v", err)
		} else {
			log.Printf("Successfully uploaded %s to MinIO", objectName)
		}
	}
}

func GenMinioCompleteObjectName(namespace, name, kind string) string {
	return fmt.Sprintf("%s/%s_%s_%s.yaml", namespace, name, kind, time.Now().Format("2006-01-02_150405"))
}
