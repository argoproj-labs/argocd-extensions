module github.com/argoproj/argocd-extensions

go 1.16

require (
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/go-getter v1.6.2
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.1
)

replace gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.1
