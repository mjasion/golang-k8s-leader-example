package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestUpdatePodLabel(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	os.Setenv("HOSTNAME", "test-pod")

	// Create a test pod
	pod := createTestPod()

	// Add the pod to the fake clientset
	_, err := clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err, "Failed to create test pod")

	// Update the pod label
	updatePodLabel(clientset)

	// Retrieve the updated pod
	updatedPod, err := clientset.CoreV1().Pods("default").Get(context.TODO(), "test-pod", metav1.GetOptions{})
	assert.NoError(t, err, "Failed to get updated pod")

	// Verify the updated label
	assert.Equal(t, "test-pod", updatedPod.Labels["pod"], "Incorrect pod label")
}

func TestUpdateServiceSelectorToCurrentPod(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	os.Setenv("HOSTNAME", "test-pod")

	// Create a test service
	service := createTestService()

	// Add the service to the fake clientset
	_, err := clientset.CoreV1().Services("default").Create(context.TODO(), service, metav1.CreateOptions{})
	assert.NoError(t, err, "Failed to create test service")

	// Update the service selector
	updateServiceSelectorToCurrentPod(clientset)

	// Retrieve the updated service
	updatedService, err := clientset.CoreV1().Services("default").Get(context.TODO(), "k8s-leader-example", metav1.GetOptions{})
	assert.NoError(t, err, "Failed to get updated service")

	// Verify the updated selector
	assert.Equal(t, "test-pod", updatedService.Spec.Selector["pod"], "Incorrect service selector")
}

func TestStatusPageHandler(t *testing.T) {
	// Create a test HTTP request
	req := httptest.NewRequest("GET", "/", nil)

	// Create a test HTTP response recorder
	res := httptest.NewRecorder()

	// Create a Gin router
	router := gin.Default()
	router.GET("/", rootPage())

	// Serve the HTTP request
	router.ServeHTTP(res, req)

	// Verify the HTTP response status code
	assert.Equal(t, http.StatusOK, res.Code, "Incorrect status code")

	// Verify the response body
	expectedBody := fmt.Sprintf(`{"hostname":"%s"}`, os.Getenv("HOSTNAME"))
	assert.Equal(t, expectedBody, res.Body.String(), "Incorrect response body")
}

func createTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels:    map[string]string{},
		},
	}
}

func createTestService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-leader-example",
			Namespace: "default",
			Labels:    map[string]string{},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{},
		},
	}
}
