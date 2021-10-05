module github.com/gardener/gardenlogin-controller-manager/.landscaper/container

go 1.16

require (
	github.com/gardener/component-cli v0.29.0
	github.com/gardener/component-spec/bindings-go v0.0.56
	github.com/gardener/gardener v1.31.0
	github.com/gardener/landscaper/apis v0.13.0
	github.com/golang/mock v1.6.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.0
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/component-base v0.21.2
	sigs.k8s.io/controller-runtime v0.9.1
	sigs.k8s.io/kind v0.7.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/gardener/gardener-resource-manager/api => github.com/gardener/gardener-resource-manager/api v0.25.0
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	google.golang.org/grpc => google.golang.org/grpc v1.27.1
	k8s.io/client-go => k8s.io/client-go v0.21.2
)
