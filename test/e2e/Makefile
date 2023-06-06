LOCAL_IMAGE_REPO	?= localhost:5001
GIT_IMAGE_NAME		?= git-http
GIT_IMAGE_TAG		?= latest

ARGOCD_EXTENSIONS_IMAGE_NAME	?= argocd-extensions
ARGOCD_IMAGE_TAG				?= test-image

GIT_IMAGE_SLUG		:= $(LOCAL_IMAGE_REPO)/$(GIT_IMAGE_NAME):$(GIT_IMAGE_TAG)
ARGOCD_EXTENSIONS_IMAGE_SLUG		:= $(LOCAL_IMAGE_REPO)/$(ARGOCD_EXTENSIONS_IMAGE_NAME):$(ARGOCD_IMAGE_TAG)

git-container-build: copy-testdata
	docker build -t $(GIT_IMAGE_SLUG) -f ./containers/git/Dockerfile ./containers/git

git-container-push: git-container-build
	docker push $(GIT_IMAGE_SLUG)

argocd-extentions-container-build:
	docker build -t $(ARGOCD_EXTENSIONS_IMAGE_SLUG) -f ../../Dockerfile ../../

argocd-extensions-container-push: argocd-extentions-container-build
	docker push $(ARGOCD_EXTENSIONS_IMAGE_SLUG)

copy-testdata:
	cp -r testdata containers/git

install-prereqs:
	assets/kind-with-registry.sh
	make argocd-extensions-container-push
	kubectl create ns argocd-extentions-e2e
	export KUBECONFIG=~/.kube/config-kind
	kubectl config set-context --current --namespace=argocd-extentions-e2e
	kustomize build prereqs/argocd | kubectl -n argocd-extentions-e2e apply -f -
	sleep 5
	make git-container-push
	kubectl apply -n argocd-extentions-e2e -f prereqs/repo/install.yaml

.PHONY: git-container-build
.PHONY: git-container-push
.PHONY: copy-testdata

clean-up:
	kind delete cluster; rm -f ~/.kube/config-kind
	docker rm -f kind-control-plane
	docker rm -f kind-registry
