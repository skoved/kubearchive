// Copyright KubeArchive Authors
// SPDX-License-Identifier: Apache-2.0
//go:build integration

package main

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"text/template"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/kubearchive/kubearchive/test"
)

const (
	numEvents  = 100
	numWorkers = 3
)

func TestDelete(t *testing.T) {
	namespaceName, _ := test.CreateTestNamespace(t, false)
	clientset, _ := test.GetKubernetesClient(t)
	sinkPort := test.PortForwardSink(t, clientset)

	resources := map[string]any{
		"resources": []map[string]any{
			{
				"selector": map[string]string{
					"apiVersion": "v1",
					"kind":       "Pod",
				},
				"deleteWhen": "true",
			},
		},
	}

	test.CreateKAC(t, namespaceName, resources)

	type PodData struct {
		Kind            string
		Version         string
		PodName         string
		PodUuid         string
		OwnerUuid       string
		CreateTimestamp string
		UpdateTimestamp string
		DeleteTimestamp string
		ResourceVersion int
		Namespace       string
	}

	podData := PodData{
		Kind:            "Pod",
		Version:         "v1",
		PodName:         "pod",
		PodUuid:         uuid.NewString(),
		OwnerUuid:       uuid.NewString(),
		CreateTimestamp: time.Now().Format(time.RFC3339),
		UpdateTimestamp: time.Now().Format(time.RFC3339),
		DeleteTimestamp: time.Now().Format(time.RFC3339),
		ResourceVersion: 1,
		Namespace:       namespaceName,
	}
	tmpl, err := template.New("pod.json").ParseFiles("testdata/pod.json")
	if err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	err = tmpl.Execute(&data, podData)
	if err != nil {
		t.Fatal(err)
	}

	event := createEvent(data.Bytes())
	wg := &sync.WaitGroup{}

	t.Log("finished setup")

	start := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go sendDeleteEvents(t, wg, sinkPort, event)
	}

	wg.Wait()

	runTime := time.Since(start)
	t.Logf("took %s to send (and the sink to process) %d cloud events", runTime.String(), numEvents*numWorkers)

	// wait the delay time so the sink executes all of the delete requests
	t.Cleanup(func() { time.Sleep(6 * time.Second) })
}

func createEvent(data []byte) cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetSpecVersion(cloudevents.VersionV1)
	event.SetSource("localhost:443")
	event.SetType("dev.knative.apiserver.resource.update")
	event.SetData(cloudevents.ApplicationJSON, data)
	return event
}

func sendDeleteEvents(t testing.TB, wg *sync.WaitGroup, sinkPort string, event cloudevents.Event) {
	p, err := cloudevents.NewHTTP()
	if err != nil {
		t.Fatal(err)
	}
	client, err := cloudevents.NewClient(p, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf("http://localhost:%s/", sinkPort)
	ctx := cloudevents.ContextWithTarget(context.Background(), url)

	for i := 0; i < numEvents; i++ {
		if result := client.Send(ctx, event); !cloudevents.IsACK(result) {
			wg.Done()
			t.Fatalf("failed to send event: %v", result)
		}
	}
	wg.Done()
}
