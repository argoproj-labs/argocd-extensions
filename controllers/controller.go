package controllers

import (
	"context"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	extensionv1 "github.com/argoproj/argocd-extensions/api/v1alpha1"
	"github.com/argoproj/argocd-extensions/pkg/extension"
)

const (
	finalizerName = "extensions-finalizer.argocd.argoproj.io"
)

// ArgoCDExtensionReconciler reconciles a ArgoCDExtension object
type ArgoCDExtensionReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	ExtensionsPath string
}

func findIndex(in []string, item string) int {
	for i := range in {
		if in[i] == item {
			return i
		}
	}
	return -1
}

func (r *ArgoCDExtensionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var original extensionv1.ArgoCDExtension
	if err := r.Get(ctx, req.NamespacedName, &original); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ext := original.DeepCopy()

	extensionCtx := extension.NewExtensionContext(ext, r.ExtensionsPath)

	if index := findIndex(ext.Finalizers, finalizerName); index > -1 && ext.DeletionTimestamp != nil {
		if err := extensionCtx.ProcessDeletion(); err != nil {
			return ctrl.Result{}, err
		}
		ext.Finalizers = append(ext.Finalizers[:index], ext.Finalizers[index+1:]...)
		err := r.Client.Update(ctx, ext)
		return ctrl.Result{}, err
	}

	readyCondition := extensionv1.ArgoCDExtensionCondition{Type: extensionv1.ConditionReady}
	if err := extensionCtx.Process(ctx); err != nil {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = err.Error()
	} else {
		readyCondition.Status = metav1.ConditionTrue
		readyCondition.Message = fmt.Sprintf("Successfully processed %d extension sources", len(original.Spec.Sources))
	}
	ext.Status.Conditions = []extensionv1.ArgoCDExtensionCondition{readyCondition}
	if !reflect.DeepEqual(ext.Status, original.Status) {
		err := r.Client.Patch(ctx, ext, client.MergeFrom(&original))
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoCDExtensionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&extensionv1.ArgoCDExtension{}).
		Complete(r)
}
