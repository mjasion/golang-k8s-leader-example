package main

import (
	"context"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	coordv1 "k8s.io/api/coordination/v1"
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
	clientset := getKubeConfig()
	updatePodLabel(clientset)
	// Create the lease object for leader election
	lease := &coordv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: leaseNamespace,
		},
		Spec: coordv1.LeaseSpec{
			LeaseDurationSeconds: pointerToInt32(10),
			HolderIdentity:       pointerToString(os.Getenv("HOSTNAME")),
		},
	}

	// Create a leaderElectionConfig for leader election
	leaderElectionConfig := leaderelection.LeaderElectionConfig{
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
		LeaseDuration: time.Duration(15) * time.Second,
		RenewDeadline: time.Duration(10) * time.Second,
		RetryPeriod:   time.Duration(5) * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: onStartedLeading,
			OnStoppedLeading: onStoppedLeading,
		},
		ReleaseOnCancel: true,
	}

	log.Println(lease.Spec.HolderIdentity)

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
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"hostname": os.Getenv("HOSTNAME"),
		})
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")

	cancel()

	wg.Wait()
}

func updatePodLabel(clientset *kubernetes.Clientset) {

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

func getKubeConfig() *kubernetes.Clientset {
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

func onStartedLeading(ctx context.Context) {
	log.Println("Became leader")
	updateService()
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

func updateService() {
	clientset := getKubeConfig()
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

func onStoppedLeading() {
	log.Println("Stopped being leader")
}

func pointerToInt32(i int32) *int32 {
	return &i
}

func pointerToString(s string) *string {
	return &s
}
