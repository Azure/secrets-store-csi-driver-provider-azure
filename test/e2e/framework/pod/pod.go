//go:build e2e
// +build e2e

package pod

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
)

// ListInput is the input of List.
type ListInput struct {
	Lister    framework.Lister
	Namespace string
	Labels    map[string]string
}

// List returns a list of pods based on labels.
func List(input ListInput) *corev1.PodList {
	Expect(input.Lister).NotTo(BeNil(), "input.Lister is required for Pod.List")
	Expect(input.Namespace).NotTo(BeNil(), "input.Namespace is required for Pod.List")
	Expect(len(input.Labels) == 0).NotTo(BeTrue(), "input.Labels is required for Pod.List")

	By(fmt.Sprintf("Listing pods with labels %v in %s namespace", input.Labels, input.Namespace))

	pods := &corev1.PodList{}
	Expect(input.Lister.List(context.TODO(), pods, client.InNamespace(input.Namespace), client.MatchingLabels(input.Labels))).Should(Succeed())

	return pods
}

// CreateInput is the input for Create.
type CreateInput struct {
	Creator                  framework.Creator
	Config                   *framework.Config
	Name                     string
	Namespace                string
	SecretProviderClassName  string
	NodePublishSecretRefName string
	Labels                   map[string]string
	ServiceAccountName       string
}

// Create creates a Pod resource.
func Create(input CreateInput) *corev1.Pod {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for Pod.Create")
	Expect(input.Config).NotTo(BeNil(), "input.Config is required for Pod.Create")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for Pod.Create")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for Pod.Create")
	Expect(input.SecretProviderClassName).NotTo(BeEmpty(), "input.SecretProviderClassName is required for Pod.Create")

	By(fmt.Sprintf("Creating Pod \"%s\"", input.Name))

	readOnly := true
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
			Labels:    input.Labels,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: to.Int64Ptr(int64(0)),
			Containers: []corev1.Container{
				{
					Name:            "tester",
					Image:           "registry.k8s.io/e2e-test-images/busybox:1.29-4",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sleep", "10000"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "secrets-store-inline",
							MountPath: "/mnt/secrets-store",
							ReadOnly:  true,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "secrets-store-inline",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							Driver:           "secrets-store.csi.k8s.io",
							ReadOnly:         &readOnly,
							VolumeAttributes: map[string]string{"secretProviderClass": input.SecretProviderClassName},
						},
					},
				},
			},
		},
	}

	if input.NodePublishSecretRefName != "" {
		for idx := range pod.Spec.Volumes {
			pod.Spec.Volumes[idx].CSI.NodePublishSecretRef = &corev1.LocalObjectReference{Name: input.NodePublishSecretRefName}
		}
	}

	if input.Config.IsWindowsTest {
		pod.Spec.NodeSelector = map[string]string{"kubernetes.io/os": "windows"}
	} else if input.Config.IsGPUTest {
		pod.Spec.NodeSelector = map[string]string{
			"kubernetes.io/os": "linux",
			"accelerator":      "nvidia",
		}
	} else {
		pod.Spec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}
	}

	if input.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = input.ServiceAccountName
	}

	Expect(input.Creator.Create(context.TODO(), pod)).Should(Succeed())
	return pod
}

// DeleteInput is the input for Delete.
type DeleteInput struct {
	Deleter framework.Deleter
	Pod     *corev1.Pod
}

// Delete deletes a pod resource.
func Delete(input DeleteInput) {
	Expect(input.Deleter).NotTo(BeNil(), "input.Deleter is required for Pod.Delete")
	Expect(input.Pod).NotTo(BeNil(), "input.Pod is required for Pod.Delete")

	By(fmt.Sprintf("Deleting Pod \"%s\"", input.Pod.Name))
	Expect(input.Deleter.Delete(context.TODO(), input.Pod)).Should(Succeed())
}

// WaitForInput is the input for WaitFor.
type WaitForInput struct {
	Getter         framework.Getter
	Config         *framework.Config
	KubeconfigPath string
	PodName        string
	Namespace      string
}

// WaitFor waits for pod to be running.
func WaitFor(input WaitForInput) {
	Expect(input.Getter).NotTo(BeNil(), "input.Getter is required for Pod.WaitFor")
	Expect(input.Config).NotTo(BeNil(), "input.Config is required for Pod.WaitFor")
	Expect(input.KubeconfigPath).NotTo(BeEmpty(), "input.KubeconfigPath is required for Pod.WaitFor")
	Expect(input.PodName).NotTo(BeEmpty(), "input.PodName is required for Pod.WaitFor")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for Pod.WaitFor")

	By(fmt.Sprintf("Ensuring Pod \"%s\" is Running", input.PodName))
	Eventually(func() bool {
		pod := &corev1.Pod{}
		Expect(input.Getter.Get(context.TODO(), client.ObjectKey{Name: input.PodName, Namespace: input.Namespace}, pod)).Should(Succeed())

		if pod.Status.Phase == corev1.PodRunning {
			return true
		}
		return false
	}, framework.Timeout, framework.Polling).Should(BeTrue())
}
