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

const sinkPort = "8111"

func BenchmarkDelete(b *testing.B) {
	namespaceName, _ := test.CreateTestNamespace(b, false)

	b.Log("created namespace")

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

	b.Log("creating kubearchiveconfig")

	test.CreateKAC(b, namespaceName, resources)

	b.Log("created kubearchiveconfig")

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
		b.Fatal(err)
	}
	var data bytes.Buffer
	err = tmpl.Execute(&data, podData)
	if err != nil {
		b.Fatal(err)
	}

	event := createEvent(data.Bytes())
	wg := &sync.WaitGroup{}

	b.Log("finished setup")

	for b.Loop() {
		wg.Add(1)
		go sendDeleteEvents(b, wg, sinkPort, event)

		wg.Add(1)
		go sendDeleteEvents(b, wg, sinkPort, event)

		wg.Add(1)
		go sendDeleteEvents(b, wg, sinkPort, event)

		wg.Wait()

	}

	b.Cleanup(func() { time.Sleep(6 * time.Second) })
}

func createEvent(data []byte) cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetSpecVersion(cloudevents.VersionV1)
	event.SetSource("localhost:443")
	event.SetType("dev.knative.apiserver.resource.update")
	event.SetData(cloudevents.ApplicationJSON, data)
	return event
}

const numEvents = 100

func sendDeleteEvents(t testing.TB, wg *sync.WaitGroup, sinkPort string, event cloudevents.Event) {
	t.Log("sending cloudevents")
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
		t.Log("sending event", i)
		if result := client.Send(ctx, event); !cloudevents.IsACK(result) {
			wg.Done()
			t.Fatalf("failed to send event: %v", result)
		}
	}
	wg.Done()
}
