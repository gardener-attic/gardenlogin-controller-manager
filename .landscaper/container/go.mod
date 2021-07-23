module github.com/gardener/gardenlogin-controller-manager/.landscaper/container

go 1.16

require (
	github.com/gardener/component-cli v0.19.0
	github.com/gardener/component-spec/bindings-go v0.0.36
	github.com/gardener/gardener v1.24.0
	github.com/gardener/landscaper/apis v0.8.4
	github.com/ghodss/yaml v1.0.0
	github.com/golang/mock v1.5.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.5
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	k8s.io/apiextensions-apiserver v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/component-base v0.20.7
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

replace k8s.io/client-go => k8s.io/client-go v0.20.2
