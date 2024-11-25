package main

import (
	//"context"
	"flag"
	"fmt"
	"k8s.io/client-go/rest"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

func main() {
	var kubeconfig string
	// 如果指定了 --kubeconfig 参数，则优先使用；否则尝试使用集群内的配置
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	var config *rest.Config
	var err error
	if kubeconfig != "" {
		// 从指定的 kubeconfig 文件加载配置
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// 使用集群内的默认配置
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Create SharedInformerFactory
	factory := informers.NewSharedInformerFactory(clientset, time.Minute)
	podInformer := factory.Core().V1().Pods().Informer()

	// Create workqueue
	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Pods")

	indexer := podInformer.GetIndexer()

	// Add event handlers to the Informer
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				fmt.Printf("Pod added: %s\n", key)
				queue.Add(key)
			}

			indexer.Add(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				fmt.Printf("Pod updated: %s\n", key)
				queue.Add(key)
			}

			indexer.Update(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				fmt.Printf("Pod deleted: %s\n", key)
				queue.Add(key)
			}

			indexer.Delete(obj)
		},
	})

	// Start the Informer
	stopCh := make(chan struct{})
	defer close(stopCh)
	go factory.Start(stopCh)

	// Wait for cache synchronization
	if !cache.WaitForCacheSync(stopCh, podInformer.HasSynced) {
		fmt.Println("Failed to sync cache")
		return
	}

	// Start the worker
	go wait.Until(func() {
		for processNextItem(queue, indexer) {
		}
	}, time.Second, stopCh)

	// Run until stopped
	<-stopCh
}

func processNextItem(queue workqueue.RateLimitingInterface, indexer cache.Indexer) bool {
	key, shutdown := queue.Get()
	if shutdown {
		return false
	}
	defer queue.Done(key)

	// Process the key (for demo purposes, just print it)
	err := handleEvent(key.(string), indexer)
	if err != nil {
		queue.AddRateLimited(key)
	} else {
		queue.Forget(key)
	}
	return true
}

func handleEvent(key string, indexer cache.Indexer) error {
	// Use the Indexer to fetch the object by key
	obj, exists, err := indexer.GetByKey(key)
	if err != nil {
		fmt.Printf("Error fetching object with key %s from indexer: %v\n", key, err)
		return err
	}

	if !exists {
		fmt.Printf("Pod %s no longer exists in the indexer\n", key)
		return nil
	}

	// Convert the object to a Pod and print its name
	pod := obj.(*v1.Pod)
	fmt.Printf("Processing Pod: %s/%s\n", pod.Namespace, pod.Name)
	return nil
}
