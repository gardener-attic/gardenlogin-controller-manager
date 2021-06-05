module github.com/gardener/gardenlogin-controller-manager

go 1.16

require (
	github.com/gardener/gardener v1.23.1
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.5
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
)

replace k8s.io/client-go => k8s.io/client-go v0.20.6
