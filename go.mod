module github.com/gardener/gardenlogin-controller-manager

go 1.16

require (
	github.com/Masterminds/semver v1.5.0
	github.com/frankban/quicktest v1.13.0 // indirect
	github.com/gardener/gardener v1.31.0
	github.com/go-logr/logr v0.4.0
	github.com/golang/snappy v0.0.3 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/apiserver v0.21.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.9.1
)

replace (
	github.com/gardener/gardener-resource-manager/api => github.com/gardener/gardener-resource-manager/api v0.25.0
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	google.golang.org/grpc => google.golang.org/grpc v1.27.1
	k8s.io/client-go => k8s.io/client-go v0.21.2
)
