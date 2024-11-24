package main

import (
	//"context"
	"flag"
	"fmt"
	"k8s.io/client-go/rest"
	"time"

	//v1 "k8s.io/api/core/v1"
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

	// Add event handlers to the Informer
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				fmt.Printf("Pod added: %s\n", key)
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				fmt.Printf("Pod updated: %s\n", key)
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				fmt.Printf("Pod deleted: %s\n", key)
				queue.Add(key)
			}
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
		for processNextItem(queue) {
		}
	}, time.Second, stopCh)

	// Run until stopped
	<-stopCh
}

func processNextItem(queue workqueue.RateLimitingInterface) bool {
	key, shutdown := queue.Get()
	if shutdown {
		return false
	}
	defer queue.Done(key)

	// Process the key (for demo purposes, just print it)
	err := handleEvent(key.(string))
	if err != nil {
		queue.AddRateLimited(key)
	} else {
		queue.Forget(key)
	}
	return true
}

func handleEvent(key string) error {
	fmt.Printf("Processing key: %s\n", key)
	return nil
}
