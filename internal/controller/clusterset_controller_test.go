/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1 "github.com/crenshaw-dev/cluster-set/api/v1alpha1"
)

var _ = Describe("ClusterSet Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		clusterset := &v1alpha1.ClusterSet{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind ClusterSet")
			err := k8sClient.Get(ctx, typeNamespacedName, clusterset)
			if err != nil && errors.IsNotFound(err) {
				resource := &v1alpha1.ClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: v1alpha1.ClusterSetSpec{
						Generator: v1alpha1.ClusterSetGenerator{
							List: &v1alpha1.ListGenerator{
								Elements: []apiextensionsv1.JSON{
									{
										Raw: []byte(`{"server": "https://example.com"}`),
									},
								},
							},
						},
						Template: v1alpha1.ClusterTemplate{
							Metadata: v1alpha1.ClusterTemplateMetadata{
								Name: resourceName,
							},
							Spec: v1alpha1.ClusterTemplateSpec{
								Server: "{{ .server }}",
							},
						},
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &v1alpha1.ClusterSet{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ClusterSet")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ClusterSetReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
			Eventually(func(g Gomega) {
				secret := &corev1.Secret{}
				err = k8sClient.Get(ctx, typeNamespacedName, secret)
				g.Expect(err).NotTo(HaveOccurred())
				// https://example.com encodes to aHR0cHM6Ly9leGFtcGxlLmNvbQ==
				g.Expect(string(secret.Data["server"])).To(Equal("aHR0cHM6Ly9leGFtcGxlLmNvbQ=="))
			}).Should(Succeed())
		})
	})
})
