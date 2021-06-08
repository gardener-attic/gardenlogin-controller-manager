module github.com/gardener/gardenlogin-controller-manager

go 1.16

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/frankban/quicktest v1.13.0 // indirect
	github.com/gardener/gardener v1.24.0
	github.com/go-logr/logr v0.4.0
	github.com/golang/snappy v0.0.3 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.5
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	k8s.io/client-go => k8s.io/client-go v0.20.6
	sigs.k8s.io/controller-runtime => github.com/gardener/controller-runtime v0.8.3-gardener.1
)
