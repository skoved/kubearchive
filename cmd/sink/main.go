// Copyright KubeArchive Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	ceOtelObs "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	ceClient "github.com/cloudevents/sdk-go/v2/client"
	"github.com/kubearchive/kubearchive/cmd/sink/expr"
	"github.com/kubearchive/kubearchive/pkg/database"
	"github.com/kubearchive/kubearchive/pkg/models"
	kaObservability "github.com/kubearchive/kubearchive/pkg/observability"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

const (
	mountPathEnvVar = "MOUNT_PATH"
)

type Sink struct {
	Db          database.DBInterface
	EventClient ceClient.Client
	Filters     *expr.Filters
	K8sClient   *dynamic.DynamicClient
	Logger      *log.Logger
}

func NewSink(db database.DBInterface, logger *log.Logger, filters *expr.Filters) *Sink {
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds|log.LUTC)
		logger.Println("sink was provided a nil logger, falling back to default logger")
	}
	if db == nil {
		logger.Fatalln("cannot start sink when db connection is nil")
	}
	if filters == nil {
		logger.Println("sink was provided nil filters. Archive and delete operations will not happen until new filters are provided")
		filters = expr.EmptyFilters()
	}

	httpClient, err := cloudevents.NewHTTP(
		cloudevents.WithRoundTripper(otelhttp.NewTransport(http.DefaultTransport)),
		cloudevents.WithMiddleware(func(next http.Handler) http.Handler {
			return otelhttp.NewHandler(next, "receive")
		}),
	)
	if err != nil {
		logger.Fatalf("failed to create HTTP client: %s\n", err.Error())
	}
	eventClient, err := cloudevents.NewClient(httpClient, ceClient.WithObservabilityService(ceOtelObs.NewOTelObservabilityService()))
	if err != nil {
		logger.Fatalf("failed to create CloudEvents HTTP client: %s\n", err.Error())
	}

	k8sClient, err := GetKubernetesClient()
	if err != nil {
		logger.Fatalln("Could not start a kubernetes client:", err)
	}

	return &Sink{
		Db:          db,
		EventClient: eventClient,
		Filters:     filters,
		K8sClient:   k8sClient,
		Logger:      logger,
	}
}

// Processes incoming cloudevents and writes them to the database
func (sink *Sink) Receive(ctx context.Context, event cloudevents.Event) {
	sink.Logger.Println("received CloudEvent: ", event.ID())
	k8sObj, err := models.UnstructuredFromByteSlice(event.Data())
	if err != nil {
		sink.Logger.Printf("cloudevent %s is malformed and will not be processed: %s\n", event.ID(), err)
		return
	}
	sink.Logger.Printf(
		"cloudevent %s contains all required fields. Checking its object %s needs to be archived\n",
		event.ID(),
		k8sObj.GetUID(),
	)
	if !sink.Filters.MustArchive(ctx, k8sObj) {
		sink.Logger.Printf("object %s from cloudevent %s does not need to be archived\n", k8sObj.GetUID(), event.ID())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	sink.Logger.Printf("writing object %s from cloudevent %s into the database\n", k8sObj.GetUID(), event.ID())
	err = sink.Db.WriteResource(ctx, k8sObj, event.Data())
	defer cancel()
	if err != nil {
		sink.Logger.Printf(
			"failed to write object %s from cloudevent %s to the database: %s\n",
			k8sObj.GetUID(),
			event.ID(),
			err,
		)
		return
	}
	sink.Logger.Printf("successfully wrote object %s cloudevent %s to the database\n", k8sObj.GetUID(), event.ID())
	sink.Logger.Printf("checking if object %s from cloudevent %s needs to be deleted\n", k8sObj.GetUID(), event.ID())
	if sink.Filters.MustDelete(ctx, k8sObj) {
		sink.Logger.Println("attempting to delete kubernetes object:", k8sObj.GetUID())
		kind := k8sObj.GetObjectKind().GroupVersionKind()
		resource, _ := meta.UnsafeGuessKindToResource(kind) // we only need the plural resource
		k8sCtx, k8sCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer k8sCancel()
		err = sink.K8sClient.Resource(resource).Namespace(k8sObj.GetNamespace()).Delete(
			k8sCtx,
			k8sObj.GetName(),
			metav1.DeleteOptions{},
		)
		if err != nil {
			sink.Logger.Printf("could not delete object %s: %s\n", k8sObj.GetUID(), err)
			return
		}
		sink.Logger.Println("successfully deleted kubernetes object:", k8sObj.GetUID())
	} else {
		sink.Logger.Printf("object %s from cloudevent %s does not need to be deleted\n", k8sObj.GetUID(), event.ID())
	}
}

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds|log.LUTC)
	err := kaObservability.Start()
	if err != nil {
		logger.Printf("Could not start tracing: %s\n", err.Error())
	}
	db, err := database.NewDatabase()
	if err != nil {
		logger.Fatalf("Could not connect to the database: %s\n", err)
	}
	filters, err := expr.NewFilters(os.Getenv(mountPathEnvVar))
	if err != nil {
		log.Printf(
			"Not all filters could be created from the ConfigMap. Some archive and delete operations will not execute until the errors are resolved: %s\n",
			err,
		)
	}
	sink := NewSink(db, logger, filters)
	err = sink.EventClient.StartReceiver(context.Background(), sink.Receive)
	if err != nil {
		logger.Fatalf("failed to start receiving CloudEvents: %s\n", err.Error())
	}
}
