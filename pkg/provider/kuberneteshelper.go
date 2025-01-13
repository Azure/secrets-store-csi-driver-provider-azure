package provider

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

const (
	// ServiceAccountClientID annotation
	ServiceAccountClientID = "azure.workload.identity/client-id"
)

type KubernetesHelper struct {
	nameSpace, svcAcc string
	k8sClient         k8sv1.CoreV1Interface
}

func NewKubernetesHelper(nameSpace, svcAcc string) KubernetesHelper {
	k8sClient, err := getK8sClient()
	if err != nil {
		panic(err)
	}

	return KubernetesHelper{
		nameSpace:  nameSpace,
		svcAcc:     svcAcc,
		k8sClient:  k8sClient,
	}
}

func (k KubernetesHelper) GetServiceAccountClientID() (string, error) {
	serviceAccount, err := k.k8sClient.ServiceAccounts(k.nameSpace).Get(context.Background(), k.svcAcc, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service account %s in namespace %s, error: %w", k.svcAcc, k.nameSpace, err)
	}
	clientID, ok := serviceAccount.Annotations[ServiceAccountClientID]
	if !ok {
		return "", fmt.Errorf("clientID not found in service account %s in namespace %s", k.svcAcc, k.nameSpace)
	}
	return clientID, nil
}

func getK8sClient() (k8sv1.CoreV1Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in cluster config, error: %w", err)
	}
	k8sClient, err := k8sv1.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return k8sClient, nil
}