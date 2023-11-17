package extension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	extensionv1 "github.com/argoproj/argocd-extensions/api/v1alpha1"
)

func NewExtensionContextWithMocks() *extensionContext {

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: "argocd",
		},
		Data: map[string][]byte{
			"sshkey": []byte("LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUFNd0FBQUF0emMyZ3RaVwpReU5UVXhPUUFBQUNCeFBxMHE0U1c5RGJmY0oxL09jdE9RajZ2NDNaTUpVckZqc1JXYTJMaVFsQUFBQUtCbUJERlZaZ1F4ClZRQUFBQXR6YzJndFpXUXlOVFV4T1FBQUFDQnhQcTBxNFNXOURiZmNKMS9PY3RPUWo2djQzWk1KVXJGanNSV2EyTGlRbEEKQUFBRURCT3g5RmVIc3ZTcjBSdzhVcEIwM2VPOU8wRlN0bHNPOWRGSzZ4cGJKREUzRStyU3JoSmIwTnQ5d25YODV5MDVDUApxL2pka3dsU3NXT3hGWnJZdUpDVUFBQUFIWEp0YjNWQVVtRm9kV3h6TFUxaFkwSnZiMnN0VUhKdkxteHZZMkZzCi0tLS0tRU5EIE9QRU5TU0ggUFJJVkFURSBLRVktLS0tLQo="),
		},
	}

	configMap := &v1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-resource-override-cm", Namespace: "argocd"},
		Data:       map[string]string{"resources": "test.customhealthcheck.com/TestResource:\n  health.lua: |2\n\n    hs = {}\n    if obj.status ~= nil and obj.status.color == \"blue\" then\n      hs.status = \"Healthy\"\n      hs.message = \"Healthy\"\n      return hs\n    end\n\n    hs.status = \"Progressing\"\n    hs.message = \"Waiting\"\n    return hs\n  ignoreDifferences: |\n    jqPathExpressions: null\n    jsonPointers: null\n    managedFieldsManagers: null\ntest.green.resource.com/TestResource:\n  health.lua: |2\n\n    hs = {}\n    if obj.status ~= nil and obj.status.color == \"green\" then\n      hs.status = \"Healthy\"\n      hs.message = \"Healthy\"\n      return hs\n    end\n\n    hs.status = \"Progressing\"\n    hs.message = \"Waiting\"\n    return hs\n  ignoreDifferences: |\n    jqPathExpressions: null\n    jsonPointers: null\n    managedFieldsManagers: null\ntest.red.resource.com/TestResource:\n  health.lua: |2\n\n    hs = {}\n    if obj.status ~= nil and obj.status.color == \"red\" then\n      hs.status = \"Healthy\"\n      hs.message = \"Healthy\"\n      return hs\n    end\n\n    hs.status = \"Progressing\"\n    hs.message = \"Waiting\"\n    return hs\n  ignoreDifferences: |\n    jqPathExpressions: null\n    jsonPointers: null\n    managedFieldsManagers: null\n"},
	}

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
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(corev1.SchemeGroupVersion, secret)
	scheme.AddKnownTypes(v1.SchemeGroupVersion, configMap)
	extensionv1.AddToScheme(scheme)
	argov1alpha1.AddToScheme(scheme)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&validargocdExtension, secret, configMap).Build()

	extensionContext := NewExtensionContext(&validargocdExtension, client, "../../test/testdata/tmp/extensions")
	return extensionContext
}

func TestShouldDownload(t *testing.T) {

	allTests := []struct {
		name            string
		Revisions       []string
		sourcesSnapshot sourcesSnapshot
	}{
		{
			name:      "No Download",
			Revisions: []string{"4e22a3cb21fa447ca362a05a505a69397c8a0d44"},
			sourcesSnapshot: sourcesSnapshot{
				Revisions: []string{"4e22a3cb21fa447ca362a05a505a69397c8a0d44"},
				Files:     []string{"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"},
			},
		},
		{
			name:      "Should Download",
			Revisions: []string{"4e22a3cb21fa447ca362a05a505a69397c8a0d45"},
			sourcesSnapshot: sourcesSnapshot{
				Revisions: []string{"4e22a3cb21fa447ca362a05a505a69397c8a0d44"},
				Files:     []string{"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"},
			},
		},
	}

	for _, test := range allTests {
		t.Run(test.name, func(t *testing.T) {
			result := test.sourcesSnapshot.shouldDownload(test.Revisions)
			if test.name == "No Download" {
				assert.Equal(t, "", result)
			}
			if test.name == "Should Download" {
				assert.Equal(t, "Source #0 has changed from 4e22a3cb21fa447ca362a05a505a69397c8a0d44 to 4e22a3cb21fa447ca362a05a505a69397c8a0d45", result)
			}

		})
	}

}

func TestDeleteFiles(t *testing.T) {

	allTests := []struct {
		name            string
		tracker         *fileTracker
		files           []string
		sourcesSnapshot sourcesSnapshot
	}{
		{
			name: "validate",
			tracker: &fileTracker{
				Files: map[string]fileMetadata{
					"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua": fileMetadata{
						Owner:        "test-extension",
						ConfigMapKey: "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"},
				},
			},
			files: []string{"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"},
			sourcesSnapshot: sourcesSnapshot{
				Revisions: []string{"4e22a3cb21fa447ca362a05a505a69397c8a0d44"},
				Files:     []string{"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"},
			},
		},
	}

	extensionContext := NewExtensionContextWithMocks()
	for _, test := range allTests {
		t.Run(test.name, func(t *testing.T) {
			err := extensionContext.deleteFiles(test.tracker, test.files)
			if assert.Error(t, err) {
				assert.Equal(t, "cannot delete file \"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua\" since it is owned by \"test-valid-argocdextension\"", err.Error())
			}
		})
	}

}

func TestBuildResourceOverrideConfigMap(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	resourceOverrides, _ := extensionContext.getExtensionResourceOverrides()

	configMap, _ := extensionContext.buildResourceOverrideConfigMap(resourceOverrides)
	assert.Equal(t, "argocd-resource-override-cm", configMap.ObjectMeta.Name)
}

func TestSetResourceOverrideConfigMap(t *testing.T) {
	configMap := &v1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-resource-override-cm", Namespace: "argocd"},
		Data:       map[string]string{"resources": "test.customhealthcheck.com/TestResource:\n  health.lua: |2\n\n    hs = {}\n    if obj.status ~= nil and obj.status.color == \"blue\" then\n      hs.status = \"Healthy\"\n      hs.message = \"Healthy\"\n      return hs\n    end\n\n    hs.status = \"Progressing\"\n    hs.message = \"Waiting\"\n    return hs\n  ignoreDifferences: |\n    jqPathExpressions: null\n    jsonPointers: null\n    managedFieldsManagers: null\ntest.green.resource.com/TestResource:\n  health.lua: |2\n\n    hs = {}\n    if obj.status ~= nil and obj.status.color == \"green\" then\n      hs.status = \"Healthy\"\n      hs.message = \"Healthy\"\n      return hs\n    end\n\n    hs.status = \"Progressing\"\n    hs.message = \"Waiting\"\n    return hs\n  ignoreDifferences: |\n    jqPathExpressions: null\n    jsonPointers: null\n    managedFieldsManagers: null\ntest.red.resource.com/TestResource:\n  health.lua: |2\n\n    hs = {}\n    if obj.status ~= nil and obj.status.color == \"red\" then\n      hs.status = \"Healthy\"\n      hs.message = \"Healthy\"\n      return hs\n    end\n\n    hs.status = \"Progressing\"\n    hs.message = \"Waiting\"\n    return hs\n  ignoreDifferences: |\n    jqPathExpressions: null\n    jsonPointers: null\n    managedFieldsManagers: null\n"},
	}
	extensionContext := NewExtensionContextWithMocks()
	error := extensionContext.setResourceOverrideConfigMap(context.Background(), configMap)
	assert.Nil(t, error)
}

func TestRebuildResourceOverrideConfigMap(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	error := extensionContext.rebuildResourceOverrideConfigMap(context.Background())
	assert.Nil(t, error)
}

func TestGetSecret(t *testing.T) {

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: "argocd",
		},
		Data: map[string][]byte{
			"sshkey": []byte("LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUFNd0FBQUF0emMyZ3RaVwpReU5UVXhPUUFBQUNCeFBxMHE0U1c5RGJmY0oxL09jdE9RajZ2NDNaTUpVckZqc1JXYTJMaVFsQUFBQUtCbUJERlZaZ1F4ClZRQUFBQXR6YzJndFpXUXlOVFV4T1FBQUFDQnhQcTBxNFNXOURiZmNKMS9PY3RPUWo2djQzWk1KVXJGanNSV2EyTGlRbEEKQUFBRURCT3g5RmVIc3ZTcjBSdzhVcEIwM2VPOU8wRlN0bHNPOWRGSzZ4cGJKREUzRStyU3JoSmIwTnQ5d25YODV5MDVDUApxL2pka3dsU3NXT3hGWnJZdUpDVUFBQUFIWEp0YjNWQVVtRm9kV3h6TFUxaFkwSnZiMnN0VUhKdkxteHZZMkZzCi0tLS0tRU5EIE9QRU5TU0ggUFJJVkFURSBLRVktLS0tLQo="),
		},
	}
	extensionContext := NewExtensionContextWithMocks()
	actualSecret, error := extensionContext.GetSecret(context.Background(), extensionv1.NamespacedName{Name: "my-secret", Namespace: "argocd"})
	assert.Nil(t, error)
	assert.Equal(t, secret.ObjectMeta.Name, actualSecret.ObjectMeta.Name)
	assert.Equal(t, secret.ObjectMeta.Namespace, actualSecret.ObjectMeta.Namespace)
}

func TestProcessDeletion(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	error := extensionContext.ProcessDeletion(context.Background())
	assert.Nil(t, error)
}

func TestGetResourceOverrideForResourceDirectory(t *testing.T) {
	_, error := getResourceOverrideForResourceDirectory("../../test/testdata/tmp/extensions", "group", "test.customhealthcheck.com")
	assert.Nil(t, error)
}

func TestGetResourceOverridesForGroupDirectory(t *testing.T) {
	_, error := getResourceOverridesForGroupDirectory("../../test/testdata/tmp/extensions", "test.customhealthcheck.com")
	assert.Nil(t, error)
}

func TestGetExtensionResourceOverrides(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	_, err := extensionContext.getExtensionResourceOverrides()
	assert.Nil(t, err)
}

func TestWalkFiles(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	list, err := extensionContext.walkFiles("../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource")
	assert.Nil(t, err)
	assert.Equal(t, []string{"../../test/testdata/tmp/extensions/health.lua"}, list)
}

func TestMoveSourceFiles(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()

	tracker := &fileTracker{
		Files: map[string]fileMetadata{
			"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua": fileMetadata{
				Owner:        "test-extension",
				ConfigMapKey: "../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"},
		},
	}
	sourcesSnapshots, err := extensionContext.moveSourceFiles(tracker, []string{"4e22a3cb21fa447ca362a05a505a69397c8a0d44"}, "../../test/testdata/tmp/extensions/resources/move/test.customhealthcheck.com/TestResource")
	assert.Nil(t, err)
	assert.Equal(t, sourcesSnapshot{Revisions: []string{"4e22a3cb21fa447ca362a05a505a69397c8a0d44"}, Files: []string{"../../test/testdata/tmp/extensions/health.lua"}}, sourcesSnapshots)
}

func TestSaveSnapshot(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	err := extensionContext.saveSnapshot(sourcesSnapshot{
		Revisions: []string{"4e22a3cb21fa447ca362a05a505a69397c8a0d44"},
		Files:     []string{"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"},
	})
	assert.Nil(t, err)
}

func TestLoadSnapshot(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	actualSnapshot := extensionContext.loadSnapshot()
	assert.Equal(t, sourcesSnapshot{Revisions: []string{"4e22a3cb21fa447ca362a05a505a69397c8a0d44"}, Files: []string{"../../test/testdata/tmp/extensions/resources/test.customhealthcheck.com/TestResource/health.lua"}}, actualSnapshot)
}

func TestDeleteSnapshot(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	err := extensionContext.deleteSnapshot()
	assert.Nil(t, err)
}

func TestDownloadTo(t *testing.T) {
	extensionContext := NewExtensionContextWithMocks()
	err := extensionContext.downloadTo(context.Background(), "../../test/testdata/tmp/extensions/download")
	assert.Nil(t, err)
}
