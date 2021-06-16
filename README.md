# gardenlogin-controller-manager

[![Slack channel #gardener](https://img.shields.io/badge/slack-gardener-brightgreen.svg?logo=slack)](https://kubernetes.slack.com/messages/gardener)
[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/gardenlogin-controller-manager)](https://goreportcard.com/report/github.com/gardener/gardenlogin-controller-manager)
[![release](https://badge.fury.io/gh/gardener%2Fgardenlogin-controller-manager.svg)](https://badge.fury.io/gh/gardener%2Fgardenlogin-controller-manager)
[![reuse compliant](https://reuse.software/badge/reuse-compliant.svg)](https://reuse.software/)

The `gardenlogin-controller-manager` renders `kubeconfig`s for accessing `Shoot` clusters. The authentication to the `Shoot` cluster is handled transparently by the [gardenlogin](https://github.com/gardener/gardenlogin) `kubectl` credential plugin. See the `gardenlogin` [authentication flow](https://github.com/gardener/gardenlogin#authentication-flow) for more details.
As the `kubeconfig`s do not contain any credentials, the `gardenlogin-controller-manager` stores the `kubeconfigs` in `ConfigMap`s under the path `data.kubeconfig`. The `ConfigMap` is named `<shoot-name>.kubeconfig`.  

## Example
### Kubeconfig
A `kubeconfig` for `Shoot` clusters with `spec.kubernetes.version` >= `v1.20.0` is rendered like below. In this case the shoot reference and garden cluster identity is passed through the cluster extensions (`clusters[].cluster.extensions[]`), which is supported starting with kubectl version `v1.20.0`.

```yaml
# supported with kubectl version v1.20.0 onwards
apiVersion: v1
kind: Config
clusters:
- name: shoot--myproject--mycluster
  cluster:
    server: https://api.mycluster.myproject.example.com
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCi4uLgotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t
    extensions:
    - name: client.authentication.k8s.io/exec
      extension:
        shootRef:
          namespace: garden-myproject
          name: mycluster
        gardenClusterIdentity: landscape-dev # must match with the garden cluster identity from the config
contexts:
- name: shoot--myproject--mycluster
  context:
    cluster: shoot--myproject--mycluster
    user: shoot--myproject--mycluster
current-context: shoot--myproject--mycluster
users:
- name: shoot--myproject--mycluster
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      provideClusterInfo: true
      command: kubectl
      args:
      - garden-login
      - get-client-certificate
```

### Legacy Kubeconfig - Support `kubectl` Versions `v1.11.0` - `v1.19.x`.
For `Shoot` clusters with `spec.kubernetes.version` < `v1.20.0` a `kubeconfig` like [example/01-kubeconfig-legacy.yaml](example/01-kubeconfig-legacy.yaml) is rendered. For these `kubeconfig`s, the `gardenlogin` plugin receives the shoot reference and garden cluster identity as command line flags. This allows us to support `kubectl` versions `v1.11.0` - `v1.19.x`.
