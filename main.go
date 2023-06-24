package main

import (
	"context"
	"github.com/caitlinelfring/go-env-default"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"

	"github.com/gin-gonic/gin"
)

const (
	leaseName = "k8s-leader-example"

	lockName = "k8s-leader-example"

	leaseNamespace = "default"
)

func main() {
	leaseDuration := env.GetIntDefault("LEASE_DURATION", 15)
	renewalDeadline := env.GetInt64Default("RENEWAL_DEADLINE", 10)
	retryPeriod := env.GetIntDefault("RETRY_PERIOD", 5)

	clientset := getKubeClient()
	updatePodLabel(clientset)

	leaderElectionConfig := leaderelection.LeaderElectionConfig{
		// Create a leaderElectionConfig for leader election
		Lock: &resourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{
				Name:      lockName,
				Namespace: leaseNamespace,
			},
			Client: clientset.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: os.Getenv("HOSTNAME"),
			},
		},
		LeaseDuration: time.Duration(leaseDuration) * time.Second,
		RenewDeadline: time.Duration(renewalDeadline) * time.Second,
		RetryPeriod:   time.Duration(retryPeriod) * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: onStartedLeading,
			OnStoppedLeading: onStoppedLeading,
		},
		ReleaseOnCancel: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Start the leader election
		leaderelection.RunOrDie(ctx, leaderElectionConfig)
	}()
	r := gin.Default()
	r.GET("/", rootPage())
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")

	cancel()

	wg.Wait()
}

func rootPage() func(c *gin.Context) {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"hostname": os.Getenv("HOSTNAME"),
		})
	}
}

func updatePodLabel(clientset kubernetes.Interface) {
	// Retrieve the pod
	hostname := os.Getenv("HOSTNAME")
	pod, err := clientset.CoreV1().Pods(metav1.NamespaceDefault).Get(context.TODO(), hostname, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}

	// Update the pod label

	existingLabels := pod.ObjectMeta.Labels
	existingLabels["pod"] = os.Getenv("HOSTNAME")
	pod.ObjectMeta.Labels = existingLabels

	// Update the pod
	updatedPod, err := clientset.CoreV1().Pods(metav1.NamespaceDefault).Update(context.TODO(), pod, metav1.UpdateOptions{})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Updated pod label: %s\n", updatedPod.Labels["pod"])
}

func onStartedLeading(ctx context.Context) {
	log.Println("Became leader: ", os.Getenv("HOSTNAME"))
	clientset := getKubeClient()
	updateServiceSelectorToCurrentPod(clientset)
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Stopped leader loop")
				return
			default:
				// Perform leader tasks here
				log.Println("Performing leader tasks...")
				time.Sleep(1 * time.Second)
			}
		}
	}()
}

func updateServiceSelectorToCurrentPod(clientset kubernetes.Interface) {
	service, err := clientset.CoreV1().Services(leaseNamespace).Get(context.TODO(), leaseName, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	// Get the existing selector labels
	existingLabels := service.Spec.Selector
	existingLabels["pod"] = os.Getenv("HOSTNAME")
	service.Spec.Selector = existingLabels

	// Update the Service
	updatedService, err := clientset.CoreV1().Services(leaseNamespace).Update(context.TODO(), service, metav1.UpdateOptions{})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Updated Service: %s\n", updatedService.Name)
}

func getKubeClient() *kubernetes.Clientset {
	// Create a Kubernetes client using the current context
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	return clientset
}

func onStoppedLeading() {
	log.Println("Stopped being leader")
}
