package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	extensionv1 "github.com/argoproj/argocd-extensions/api/v1alpha1"
)

func TestReconcilerValidationErrorBehaviour(t *testing.T) {

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: "argocd",
		},
		Data: map[string][]byte{
			"sshkey": []byte("LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUFNd0FBQUF0emMyZ3RaVwpReU5UVXhPUUFBQUNCeFBxMHE0U1c5RGJmY0oxL09jdE9RajZ2NDNaTUpVckZqc1JXYTJMaVFsQUFBQUtCbUJERlZaZ1F4ClZRQUFBQXR6YzJndFpXUXlOVFV4T1FBQUFDQnhQcTBxNFNXOURiZmNKMS9PY3RPUWo2djQzWk1KVXJGanNSV2EyTGlRbEEKQUFBRURCT3g5RmVIc3ZTcjBSdzhVcEIwM2VPOU8wRlN0bHNPOWRGSzZ4cGJKREUzRStyU3JoSmIwTnQ5d25YODV5MDVDUApxL2pka3dsU3NXT3hGWnJZdUpDVUFBQUFIWEp0YjNWQVVtRm9kV3h6TFUxaFkwSnZiMnN0VUhKdkxteHZZMkZzCi0tLS0tRU5EIE9QRU5TU0ggUFJJVkFURSBLRVktLS0tLQo="),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(corev1.SchemeGroupVersion, secret)
	err := extensionv1.AddToScheme(scheme)
	assert.Nil(t, err)
	err = argov1alpha1.AddToScheme(scheme)
	assert.Nil(t, err)

	validargocdExtension := extensionv1.ArgoCDExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-valid-argocdextension",
			Namespace: "argocd",
		},
		Spec: extensionv1.ArgoCDExtensionSpec{
			Sources: []extensionv1.ExtensionSource{
				{
					Git: &extensionv1.GitSource{
						Url:      "https://github.com/argoproj/argo-cd.git",
						Revision: "HEAD",
						Secret: &extensionv1.NamespacedName{
							Name:      "my-secret",
							Namespace: "argocd",
						},
					},
				},
			},
			BaseDirectory: "resource_customizations/argoproj.io/ApplicationSet",
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&validargocdExtension, secret).Build()

	r := ArgoCDExtensionReconciler{
		Client:         client,
		Scheme:         scheme,
		ExtensionsPath: "/tmp/resource",
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "argocd",
			Name:      "test-valid-argocdextension",
		},
	}

	res, err := r.Reconcile(context.Background(), req)
	assert.Nil(t, err)
	assert.True(t, res.RequeueAfter == 0)

	var argocdExtension extensionv1.ArgoCDExtension

	// make sure argocd extension got created
	err = r.Client.Get(context.TODO(), crtclient.ObjectKey{Namespace: "argocd", Name: "test-valid-argocdextension"}, &argocdExtension)
	assert.NoError(t, err)
	assert.Equal(t, argocdExtension, "test-valid-argocdextension")
}
