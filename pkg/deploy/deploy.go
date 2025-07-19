package deploy

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	llamav1alpha1 "github.com/llamastack/llama-stack-k8s-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyDeployment creates or updates the Deployment.
func ApplyDeployment(ctx context.Context, cli client.Client, scheme *runtime.Scheme,
	instance *llamav1alpha1.LlamaStackDistribution, deployment *appsv1.Deployment, logger logr.Logger) error {
	if err := ctrl.SetControllerReference(instance, deployment, scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	found := &appsv1.Deployment{}
	err := cli.Get(ctx, client.ObjectKeyFromObject(deployment), found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating Deployment", "deployment", deployment.Name)
		return cli.Create(ctx, deployment)
	} else if err != nil {
		return fmt.Errorf("failed to fetch deployment: %w", err)
	}

	// For updates, preserve the existing selector since it's immutable
	// and use server-side apply for other fields
	if !reflect.DeepEqual(found.Spec, deployment.Spec) {
		logger.Info("Updating Deployment", "deployment", deployment.Name)

		// Preserve the existing selector to avoid immutable field error
		deployment.Spec.Selector = found.Spec.Selector

		// Use server-side apply to merge changes properly
		// Ensure the deployment has proper TypeMeta for server-side apply
		deployment.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
		return cli.Patch(ctx, deployment, client.Apply, client.ForceOwnership, client.FieldOwner("llama-stack-operator"))
	}
	return nil
}

// ApplyService creates or updates the Service.
func ApplyService(ctx context.Context, cli client.Client, scheme *runtime.Scheme,
	instance *llamav1alpha1.LlamaStackDistribution, service *corev1.Service, logger logr.Logger) error {
	if err := ctrl.SetControllerReference(instance, service, scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	found := &corev1.Service{}
	err := cli.Get(ctx, client.ObjectKeyFromObject(service), found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating Service", "service", service.Name)
		return cli.Create(ctx, service)
	} else if err != nil {
		return fmt.Errorf("failed to fetch Service: %w", err)
	}

	if !reflect.DeepEqual(found.Spec.Selector, service.Spec.Selector) || !reflect.DeepEqual(found.Spec.Ports, service.Spec.Ports) {
		found.Spec.Selector = service.Spec.Selector
		found.Spec.Ports = service.Spec.Ports
		logger.Info("Updating Service", "service", service.Name)
		return cli.Update(ctx, found)
	}
	return nil
}
