CLIENT_ID=$1
CLIENT_SECRET=$2

# create kind cluster
kind create cluster --name kind-csi-demo

# install csi-secrets-store-provider-azure
helm repo add csi-secrets-store-provider-azure https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/charts
helm install csi-secrets-store-provider-azure/csi-secrets-store-provider-azure --generate-name

# create secret in k8
kubectl create secret generic secrets-store-creds --from-literal clientid=$CLIENT_ID --from-literal clientsecret=$CLIENT_SECRET

# Deploy app
kubectl apply -f nginx-pod-secrets-store-inline-volume.yaml

# wait for deployment
kubectl wait --for=condition=ready pods/nginx-secrets-store-inline --timeout=300s

# validate
kubectl exec -it nginx-secrets-store-inline ls /mnt/secrets-store/