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
	"fmt"
	"github.com/crenshaw-dev/cluster-set/internal/controller/argocd"
	"github.com/crenshaw-dev/cluster-set/internal/generators"
	"github.com/crenshaw-dev/cluster-set/internal/template"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/crenshaw-dev/cluster-set/api/v1alpha1"
)

// ClusterSetReconciler reconciles a ClusterSet object
type ClusterSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=argoproj.io,resources=clustersets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=clustersets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=argoproj.io,resources=clustersets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *ClusterSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cs v1alpha1.ClusterSet
	if err := r.Get(ctx, req.NamespacedName, &cs); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ClusterSet not found")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch ClusterSet")
		return ctrl.Result{}, fmt.Errorf("unable to fetch ClusterSet: %w", err)
	}

	if cs.Spec.Generator.List == nil {
		// TODO: support more than just list generator
		return ctrl.Result{}, fmt.Errorf("list generator must not be nil")
	}

	paramsList, err := generators.GetParametersFromListGenerator(cs.Spec.Generator.List)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to get parameters from list generator: %w", err)
	}

	var t *v1alpha1.ClusterTemplate
	var secret *v1.Secret
	for i := range paramsList {
		t, err = template.Render(&cs.Spec.Template, paramsList[i])
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to template using params at index %d: %w", i, err)
		}
		secret, err = argocd.ClusterTemplateToSecret(*t)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to convert cluster to secret from params at index %d: %w", i, err)
		}

		// Namespace is always the same as the ClusterSet
		secret.Namespace = cs.Namespace

		err = r.Client.Get(ctx, client.ObjectKey{Namespace: secret.Namespace, Name: secret.Name}, secret)
		if err != nil {
			if errors.IsNotFound(err) {
				logger.Info("Creating secret", "name", secret.Name)
				err = r.Client.Create(ctx, secret)
				if err != nil {
					logger.Error(err, "unable to create secret")
					return ctrl.Result{}, fmt.Errorf("unable to create secret: %w", err)
				}
			} else {
				logger.Error(err, "unable to get secret")
				return ctrl.Result{}, fmt.Errorf("unable to get secret: %w", err)
			}
		} else {
			logger.Info("Updating secret", "name", secret.Name)
			err = r.Client.Update(ctx, secret)
			if err != nil {
				logger.Error(err, "unable to update secret")
				return ctrl.Result{}, fmt.Errorf("unable to update secret: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterSet{}).
		Named("clusterset").
		Complete(r)
}
