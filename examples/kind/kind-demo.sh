#!/usr/bin/env bash

CLIENT_ID=$1
CLIENT_SECRET=$2

# create kind cluster
kind create cluster --name kind-csi-demo

# install csi-secrets-store-provider-azure
helm repo add csi-secrets-store-provider-azure https://azure.github.io/secrets-store-csi-driver-provider-azure/charts
helm install csi-secrets-store-provider-azure/csi-secrets-store-provider-azure --generate-name

# create secret in k8
kubectl create secret generic secrets-store-creds --from-literal clientid="$CLIENT_ID" --from-literal clientsecret="$CLIENT_SECRET"

# label the secret
kubectl label secret secrets-store-creds secrets-store.csi.k8s.io/used=true

# Deploy app
kubectl apply -f v1alpha1_secretproviderclass.yaml
kubectl apply -f pod-secrets-store-inline-volume-secretproviderclass.yaml

# wait for deployment
kubectl wait --for=condition=ready pods/busybox-secrets-store-inline --timeout=300s

# validate
kubectl exec busybox-secrets-store-inline ls /mnt/secrets-store/
