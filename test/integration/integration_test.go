//go:build integration

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/kubearchive/kubearchive/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestMain(m *testing.M) {
	if os.Getenv("KO_DOCKER_REPO") == "" {
		os.Setenv("KO_DOCKER_REPO", "kind.local")
	}
	os.Exit(m.Run())
}

// This test is redundant with the kubectl rollout status from the hack/quick-install.sh
// but it serves as a valid integration test, not a dummy that is not testing anything real.
func TestKubeArchiveDeployments(t *testing.T) {
	client, err := test.GetKubernetesClient()
	if err != nil {
		t.Fatal(err)
	}

	retryErr := retry.Do(func() error {
		deployments, err := client.AppsV1().Deployments("kubearchive").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("Failed to get Deployments from the 'kubearchive' namespace: %w", err)
		}

		if len(deployments.Items) == 0 {
			return errors.New("No deployments found in the 'kubearchive' namespace, something went wrong.")
		}

		areAllReady := true
		for _, deployment := range deployments.Items {
			t.Logf("Deployment '%s' has '%d' ready replicas", deployment.Name, deployment.Status.ReadyReplicas)
			areAllReady = areAllReady && deployment.Status.ReadyReplicas >= 1
		}

		if areAllReady {
			t.Log("All deployments ready.")
			return nil
		}

		return errors.New("Timed out while waiting for deployments to be ready.")
	})

	if retryErr != nil {
		t.Fatal(retryErr)
	}
}

// This test verifies the database connection retries using the Sink component.
func TestDatabaseConnection(t *testing.T) {
	clientset, err := test.GetKubernetesClient()
	if err != nil {
		t.Fatal(err)
	}

	dynclient, err := test.GetDynamicKubernetesClient()
	if err != nil {
		t.Fatal(err)
	}

	// Fence database
	clusterResource := schema.GroupVersionResource{Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters"}
	resource, err := dynclient.Resource(clusterResource).Namespace("postgresql").Get(context.Background(), "kubearchive", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	annotations := resource.GetAnnotations()
	annotations["cnpg.io/fencedInstances"] = "[\"*\"]"
	resource.SetAnnotations(annotations)

	resource, err = dynclient.Resource(clusterResource).Namespace("postgresql").Update(context.Background(), resource, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Fenced database")

	// restart sink pod - replicas = 0
	deploymentScaleSink, err := clientset.AppsV1().Deployments("kubearchive").GetScale(context.Background(), "kubearchive-sink", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	scaleSink := *deploymentScaleSink
	scaleSink.Spec.Replicas = 0
	usSink, err := clientset.AppsV1().Deployments("kubearchive").UpdateScale(context.Background(), "kubearchive-sink", &scaleSink, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Changing sink to %d replicas", scaleSink.Spec.Replicas)
	t.Log(*usSink)

	t.Logf("Waiting 5 seconds for kubearchive-sink to scale down...")
	time.Sleep(5 * time.Second)

	// restart sink pod - replicas = 1
	deploymentScaleSink, err = clientset.AppsV1().Deployments("kubearchive").GetScale(context.Background(), "kubearchive-sink", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	scaleSink = *deploymentScaleSink
	scaleSink.Spec.Replicas = 1
	usSink, err = clientset.AppsV1().Deployments("kubearchive").UpdateScale(context.Background(), "kubearchive-sink", &scaleSink, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Changing sink to %d replicas", scaleSink.Spec.Replicas)
	t.Log(*usSink)

	// wait to sink pod ready and generate connection retries with the database
	t.Logf("Waiting for sink to be up, and to generate retries with the database")
	retryErr := retry.Do(func() error {
		logs, err := test.GetPodLogs(clientset, "kubearchive", "kubearchive-sink")
		if err != nil {
			return err
		}

		t.Logf("Logs\n%s", logs)
		if strings.Contains(logs, "connection refused") {
			return nil
		}

		return fmt.Errorf("Pod didn't try to connect to the database yet")
	})

	if retryErr != nil {
		t.Fatal(retryErr)
	}

	// Unfence database
	resource, err = dynclient.Resource(clusterResource).Namespace("postgresql").Get(context.Background(), "kubearchive", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	annotations = resource.GetAnnotations()
	delete(annotations, "cnpg.io/fencedInstances")
	resource.SetAnnotations(annotations)

	resource, err = dynclient.Resource(clusterResource).Namespace("postgresql").Update(context.Background(), resource, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Unfenced database")

	retryErr = retry.Do(func() error {
		logs, err := test.GetPodLogs(clientset, "kubearchive", "kubearchive-sink")
		if err != nil {
			return nil
		}

		t.Logf("Logs:\n%s", logs)
		if strings.Contains(logs, "Successfully connected to the database") {
			return nil
		}

		return errors.New("Pod didn't connect successfully to the database yet")
	})

	if retryErr != nil {
		t.Fatal(retryErr)
	}
}
