/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"syscall"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"

	extensionv1 "github.com/argoproj/argocd-extensions/api/v1"
)

// ArgoCDExtensionReconciler reconciles a ArgoCDExtension object
type ArgoCDExtensionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func Clone(path string, repo string, revision string) (ctrl.Result, error) {
	log.Infof("Cloning into %s", path)
	if _, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:           repo,
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName(revision),
	}); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return ctrl.Result{}, nil
}

//+kubebuilder:rbac:groups=extension.argoproj.io,resources=argocdextensions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extension.argoproj.io,resources=argocdextensions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=extension.argoproj.io,resources=argocdextensions/finalizers,verbs=update
func (r *ArgoCDExtensionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = k8slog.FromContext(ctx)
	var extension extensionv1.ArgoCDExtension
	if err := r.Get(ctx, req.NamespacedName, &extension); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if extension.Spec.Extends != "ui" {
		return ctrl.Result{}, nil
	}

	path := fmt.Sprintf("/tmp/extensions/%s/%s", extension.Spec.Target.Resource.Group, extension.Spec.Target.Resource.Kind)

	if extension.Spec.Source.Repository != nil {
		repoInfo := extension.Spec.Source.Repository
		repo, err := git.PlainOpen(path)

		cloneRepo := func() (ctrl.Result, error) {
			res, err := Clone(path, *repoInfo.Url, *repoInfo.Revision)
			if err != nil {
				log.Errorf("problem cloning repo %s", path)
			}
			return res, err
		}

		if err != nil {
			log.Infof("Extension directory %s does not exist, cloning now", path)
			return cloneRepo()
		}

		worktree, err := repo.Worktree()
		if err != nil {
			log.Infof("Could not open worktree for %s, cloning now", path)
			return cloneRepo()
		}

		err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil {
			log.Errorf("problem pulling remote for repo %s: %s", path, err)
		}
	}

	if url := extension.Spec.Source.File; url != nil {
		path = fmt.Sprintf("%s/ui", path)
		fullPath := fmt.Sprintf("%s/extensions.js", path)

		res, err := http.Get(*url)
		if err != nil {
			log.Errorf("problem downloading file from url %s: %s", *url, err)
		}
		defer res.Body.Close()

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			log.Infof("Creating path %s", path)
			syscall.Umask(0)
			os.MkdirAll(path, 0755)
		}

		outfile, err := os.Create(fullPath)
		if err != nil {
			log.Errorf("problem creating output file %s: %s", path, err)
		}
		defer outfile.Close()
		_, err = io.Copy(outfile, res.Body)
		if err != nil {
			log.Errorf("problem writing to output file %s: %s", path, err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoCDExtensionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&extensionv1.ArgoCDExtension{}).
		Complete(r)
}
