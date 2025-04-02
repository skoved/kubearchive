#!/usr/bin/bash
# Copyright KubeArchive Authors
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o xtrace

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)
cd ${SCRIPT_DIR}

# install strimzi
kubectl get namespace kafka &> /dev/null || kubectl create namespace kafka
kubectl get deployment strimzi-cluster-operator -n kafka &> /dev/null || kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka
kubectl rollout status deployment --namespace=kafka --timeout=100s

# create kafka cluster
kubectl apply -f https://strimzi.io/examples/latest/kafka/kraft/kafka-single-node.yaml -n kafka
kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka

# reinstall knative-eventing but modify the default broker implementation for the kubearchive namespace
kubectl apply -k base
kubectl rollout status deployment --namespace=knative-eventing --timeout=120s
kubectl apply -f kafka-broker-config.yaml

# install knative-eventing kafka controller and data plane
kubectl apply --filename https://github.com/knative-extensions/eventing-kafka-broker/releases/download/knative-v1.15.0/eventing-kafka-controller.yaml
kubectl apply --filename https://github.com/knative-extensions/eventing-kafka-broker/releases/download/knative-v1.15.0/eventing-kafka-broker.yaml
kubectl rollout status deployment --namespace=knative-eventing --timeout=120s

# delete kubearchive brokers so that they will be recreated
kubectl delete -n kubearchive brokers --all

bash ../../hack/kubearchive-install.sh
