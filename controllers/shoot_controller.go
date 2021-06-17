/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gardener/gardenlogin-controller-manager/api/v1alpha1"
	"github.com/gardener/gardenlogin-controller-manager/api/v1alpha1/constants"
	"github.com/gardener/gardenlogin-controller-manager/internal/util"

	"github.com/Masterminds/semver"
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/utils/infodata"
	"github.com/gardener/gardener/pkg/utils/secrets"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kErros "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/wait"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ShootReconciler reconciles a Shoot object
type ShootReconciler struct {
	Scheme *runtime.Scheme
	client.Client
	Log                         logr.Logger
	Config                      *util.ControllerManagerConfiguration
	ReconcilerCountPerNamespace map[string]int
	mutex                       sync.RWMutex
	configMutex                 sync.RWMutex
}

//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete;manage
//+kubebuilder:rbac:groups="",resources=configmaps/finalizers,verbs=update
//+kubebuilder:rbac:groups="core.gardener.cloud",resources=shootstates,verbs=get;list;watch;
//+kubebuilder:rbac:groups="core.gardener.cloud",resources=shoots,verbs=get;list;watch;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ShootReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("shoot", req.NamespacedName)

	if err := r.increaseCounterForNamespace(req.Namespace); err != nil {
		log.Info("maximum parallel reconciles reached for namespace - requeuing the req")

		return ctrl.Result{
			RequeueAfter: wait.Jitter(time.Duration(int64(100*time.Millisecond)), 50), // requeue after 100ms - 5s
		}, nil
	}

	res, err := r.handleRequest(ctx, req)

	r.decreaseCounterForNamespace(req.Namespace)

	return res, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShootReconciler) SetupWithManager(mgr ctrl.Manager, config util.ShootControllerConfiguration) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gardencorev1beta1.Shoot{}, builder.WithPredicates(r.shootPredicate())).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(r.configMapPredicate())).
		Watches(&source.Kind{Type: &gardencorev1alpha1.ShootState{}},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name:      o.GetName(),
							Namespace: o.GetNamespace(),
						},
					},
				}
			}),
			builder.WithPredicates(r.shootStatePredicate())).
		Named("main").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: config.MaxConcurrentReconciles,
		}).
		Complete(r)
}

// shootPredicate returns true for all create and delete events. It returns true for update events in case the advertised addresses have changed
func (r *ShootReconciler) shootPredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log := r.Log.WithValues("event", e)

			if e.ObjectOld == nil {
				log.Error(nil, "Update event has no old runtime object to update")
				return false
			}

			if e.ObjectNew == nil {
				log.Error(nil, "Update event has no new runtime object for update")
				return false
			}

			old, ok := e.ObjectOld.(*gardencorev1beta1.Shoot)
			if !ok {
				log.Error(nil, "Update event old runtime object cannot be converted to Shoot")
				return false
			}

			new, ok := e.ObjectNew.(*gardencorev1beta1.Shoot)
			if !ok {
				log.Error(nil, "Update event new runtime object cannot be converted to Shoot")
				return false
			}

			// length has changed - event should be processed
			if len(old.Status.AdvertisedAddresses) != len(new.Status.AdvertisedAddresses) {
				return true
			}

			// if the advertised addresses have changed the event should be processed
			for i, addressNew := range new.Status.AdvertisedAddresses {
				addressOld := old.Status.AdvertisedAddresses[i]
				if addressOld.Name != addressNew.Name {
					return true
				}

				if addressOld.URL != addressNew.URL {
					return true
				}
			}

			// no change detected that is relevant for this controller
			return false
		},
	}
}

// configMapPredicate returns true for all create and delete events. It returns true for update events in case the kubeconfig data or the kubeconfig role label has changed
func (r *ShootReconciler) configMapPredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log := r.Log.WithValues("event", e)

			if e.ObjectOld == nil {
				log.Error(nil, "Update event has no old runtime object to update")
				return false
			}

			if e.ObjectNew == nil {
				log.Error(nil, "Update event has no new runtime object for update")
				return false
			}

			old, ok := e.ObjectOld.(*corev1.ConfigMap)
			if !ok {
				log.Error(nil, "Update event old runtime object cannot be converted to ConfigMap")
				return false
			}

			new, ok := e.ObjectNew.(*corev1.ConfigMap)
			if !ok {
				log.Error(nil, "Update event new runtime object cannot be converted to ConfigMap")
				return false
			}

			// ignore configmaps that do not have the kubeconfig role
			if old.Labels[constants.GardenerOperationsRole] != constants.GardenerOperationsKubeconfig &&
				new.Labels[constants.GardenerOperationsRole] != constants.GardenerOperationsKubeconfig {
				return false
			}

			// handle event in case the role has changed
			if old.Labels[constants.GardenerOperationsRole] != new.Labels[constants.GardenerOperationsRole] {
				return true
			}

			// handle event in case the kubeconfig has changed
			if old.Data[constants.DataKeyKubeconfig] != new.Data[constants.DataKeyKubeconfig] {
				return true
			}

			// no change detected that is relevant for this controller
			return false
		},
	}
}

// shootStatePredicate returns true for all create and delete events. It returns true for update events in case the cluster ca changes
func (r *ShootReconciler) shootStatePredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log := r.Log // do not set the event as log.WithValues as it may contain credentials

			if e.ObjectOld == nil {
				log.Error(nil, "Update event has no old runtime object to update")
				return false
			}

			if e.ObjectNew == nil {
				log.Error(nil, "Update event has no new runtime object for update")
				return false
			}

			old, ok := e.ObjectOld.(*gardencorev1alpha1.ShootState)
			if !ok {
				log.Error(nil, "Update event old runtime object cannot be converted to ShootState")
				return false
			}

			// enhance log with name and namespace of the object for which the event occurred
			log = log.WithValues("name", old.Name, "namespace", old.Namespace)

			new, ok := e.ObjectNew.(*gardencorev1alpha1.ShootState)
			if !ok {
				log.Error(nil, "Update event new runtime object cannot be converted to ShootState")
				return false
			}

			oldCaCert, err := clusterCaCert(old)
			if err != nil && !errors.Is(err, errCaNotProvisioned) {
				log.Error(nil, "Update event failed to read cluster ca from old ShootState", "error", err)
				return false
			}

			newCaCert, err := clusterCaCert(new)
			if err != nil {
				// The errCaNotProvisioned is usually returned for newly created clusters, in this case we do not want to log it as error as it is expected.
				// However in case the new ca cert is nil, it does not make sense to handle the event and that's why we skip it
				if !errors.Is(err, errCaNotProvisioned) {
					log.Error(nil, "Update event failed to read cluster ca from new ShootState", "error", err)
				}
				return false
			}

			// if the ca cert has changed, we want to handle the event
			return !apiequality.Semantic.DeepEqual(oldCaCert, newCaCert)
		},
	}
}

func (r *ShootReconciler) handleRequest(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	kubeconfigName := fmt.Sprintf("%s.kubeconfig", req.Name)
	kubeconfigConfigMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: kubeconfigName, Namespace: req.Namespace}}

	// fetch Shoot
	shoot := &gardencorev1beta1.Shoot{}

	if err := r.Client.Get(ctx, req.NamespacedName, shoot); err != nil {
		if kErros.IsNotFound(err) {
			// shoot does not exist anymore - cleanup kubeconfig configmap
			return ctrl.Result{}, client.IgnoreNotFound(r.Client.Delete(ctx, kubeconfigConfigMap))
		}
		// Error reading the object - requeue the request
		return ctrl.Result{}, err
	}

	// fetch ShootState
	shootState := &gardencorev1alpha1.ShootState{}

	if err := r.Get(ctx, req.NamespacedName, shootState); err != nil {
		if kErros.IsNotFound(err) {
			// shootstate does not exist anymore - cleanup kubeconfig configmap
			return ctrl.Result{}, client.IgnoreNotFound(r.Client.Delete(ctx, kubeconfigConfigMap))
		}
		// Error reading the object - requeue the request
		return ctrl.Result{}, err
	}

	if shootState.DeletionTimestamp != nil {
		// shootstate is in deletion - cleanup kubeconfig configmap
		return ctrl.Result{}, client.IgnoreNotFound(r.Client.Delete(ctx, kubeconfigConfigMap))
	}

	if shoot.DeletionTimestamp != nil {
		// shoot is in deletion - cleanup kubeconfig configmap
		return ctrl.Result{}, client.IgnoreNotFound(r.Client.Delete(ctx, kubeconfigConfigMap))
	}

	if len(shoot.Status.AdvertisedAddresses) == 0 {
		return ctrl.Result{}, errors.New("no kube-apiserver advertised addresses in Shoot .status.advertisedAddresses")
	}

	caCert, err := clusterCaCert(shootState)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err = util.ValidateCertificate(caCert); err != nil {
		return ctrl.Result{}, fmt.Errorf("an error occured validating the ca certificate: %w", err)
	}

	clusterIdentityConfigMap := &corev1.ConfigMap{}
	key := types.NamespacedName{
		Name:      v1beta1constants.ClusterIdentity,
		Namespace: "kube-system",
	}

	if err = r.Client.Get(ctx, key, clusterIdentityConfigMap); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to fetch garden cluster identity: %w", err)
	}

	if clusterIdentityConfigMap.Data == nil {
		return ctrl.Result{}, errors.New("cluster identity configmap data not set")
	}

	kubeconfigRequest := kubeconfigRequest{
		namespace:             shoot.Namespace,
		shootName:             shoot.Name,
		gardenClusterIdentity: clusterIdentityConfigMap.Data[v1beta1constants.ClusterIdentity],
	}

	for _, address := range shoot.Status.AdvertisedAddresses {
		u, err := url.Parse(address.URL)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("could not parse shoot server url: %w", err)
		}

		kubeconfigRequest.clusters = append(kubeconfigRequest.clusters, cluster{
			name:          address.Name,
			apiServerHost: u.Host,
			caCert:        caCert,
		})
	}

	if err = kubeconfigRequest.validate(); err != nil {
		return ctrl.Result{}, fmt.Errorf("validation failed for kubeconfig request: %w", err)
	}

	// parse kubernetes version to determine if a legacy kubeconfig should be created.
	c, err := semver.NewConstraint("< v1.20.0")
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse constraint: %w", err)
	}

	version, err := semver.NewVersion(shoot.Spec.Kubernetes.Version)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not parse kubernetes version %s of shoot cluster: %w", shoot.Spec.Kubernetes.Version, err)
	}

	legacy := c.Check(version)

	kubeconfig, err := kubeconfigRequest.generate(legacy)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("generation failed for kubeconfig request: %w", err)
	}

	ownerReference := metav1.NewControllerRef(shoot, gardencorev1beta1.SchemeGroupVersion.WithKind("Shoot"))
	ownerReference.BlockOwnerDeletion = pointer.BoolPtr(false)

	// store the kubeconfig in a ConfigMap, as it does not contain any credentials or other secret data
	if _, err = ctrl.CreateOrUpdate(ctx, r.Client, kubeconfigConfigMap, func() error {
		kubeconfigConfigMap.OwnerReferences = []metav1.OwnerReference{*ownerReference}

		if kubeconfigConfigMap.Labels == nil {
			kubeconfigConfigMap.Labels = make(map[string]string)
		}
		kubeconfigConfigMap.Labels[constants.GardenerOperationsRole] = constants.GardenerOperationsKubeconfig

		if kubeconfigConfigMap.Data == nil {
			kubeconfigConfigMap.Data = make(map[string]string)
		}
		kubeconfigConfigMap.Data[constants.DataKeyKubeconfig] = string(kubeconfig)
		return nil
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create or update kubeconfig configmap %s/%s: %w", kubeconfigConfigMap.Namespace, kubeconfigConfigMap.Name, err)
	}

	return ctrl.Result{}, nil
}

var errCaNotProvisioned = errors.New("certificate authority not yet provisioned")

// clusterCaCert reads the ca certificate from the gardener resource data
func clusterCaCert(shootState *gardencorev1alpha1.ShootState) ([]byte, error) {
	ca, err := infodata.GetInfoData(shootState.Spec.Gardener, v1beta1constants.SecretNameCACluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get ca infoData: %w", err)
	}

	if ca == nil {
		return nil, errCaNotProvisioned
	}

	caInfoData, ok := ca.(*secrets.CertificateInfoData)
	if !ok {
		return nil, errors.New("could not convert InfoData entry ca to CertificateInfoData")
	}

	caCert := caInfoData.Certificate

	return caCert, nil
}

// kubeconfigRequest is a struct which holds information about a Kubeconfig to be generated.
type kubeconfigRequest struct {
	// cluster holds all the cluster on which the kube-apiserver can be reached
	clusters []cluster
	// namespace is the namespace where the shoot resides
	namespace string
	// shootName is the name of the shoot
	shootName string
	// gardenClusterIdentity is the cluster identifier of the garden cluster.
	gardenClusterIdentity string
}

// cluster holds the data to describe and connect to a kubernetes cluster
type cluster struct {
	// name is the name of the shoot advertised address, usually "external", "internal" or "unmanaged"
	name string
	// apiServerHost is the host of the kube-apiserver
	apiServerHost string

	// caCert holds the ca certificate for the cluster
	//+optional
	caCert []byte
}

// validate validates the kubeconfig request by ensuring that all required fields are set
func (k *kubeconfigRequest) validate() error {
	if len(k.clusters) == 0 {
		return errors.New("missing clusters")
	}

	for n, cluster := range k.clusters {
		if cluster.name == "" {
			return fmt.Errorf("no name defined for cluster[%d]", n)
		}

		if cluster.apiServerHost == "" {
			return fmt.Errorf("no api server host defined for cluster[%d]", n)
		}
	}

	if k.namespace == "" {
		return errors.New("no namespace defined for kubeconfig request")
	}

	if k.shootName == "" {
		return errors.New("no shoot name defined for kubeconfig request")
	}

	if k.gardenClusterIdentity == "" {
		return errors.New("no garden cluster identity defined for kubeconfig request")
	}

	return nil
}

// generate generates a Kubernetes kubeconfig for communicating with the kube-apiserver
// by exec'ing the gardenlogin plugin, which fetches a client certificate.
// If legacy is false, the shoot reference and garden cluster identity is passed via the cluster extensions,
// which is supported starting with kubectl version v1.20.0.
// If legacy is true, the shoot reference and garden cluster identity are passed as command line flags to the plugin
func (k *kubeconfigRequest) generate(legacy bool) ([]byte, error) {
	authName := fmt.Sprintf("%s--%s", k.namespace, k.shootName)
	name := fmt.Sprintf("%s-%s", authName, k.clusters[0].name)

	var legacyArgs []string
	if legacy {
		legacyArgs = []string{
			fmt.Sprintf("--name=%s", k.shootName),
			fmt.Sprintf("--namespace=%s", k.namespace),
			fmt.Sprintf("--garden-cluster-identity=%s", k.gardenClusterIdentity),
		}
	}

	var authInfos []clientcmdv1.NamedAuthInfo
	authInfos = append(authInfos, clientcmdv1.NamedAuthInfo{
		Name: authName,
		AuthInfo: clientcmdv1.AuthInfo{
			Exec: &clientcmdv1.ExecConfig{
				Command: "kubectl",
				Args: append([]string{
					"gardenlogin",
					"get-client-certificate",
				},
					legacyArgs...,
				),
				Env:                nil,
				APIVersion:         clientauthenticationv1beta1.SchemeGroupVersion.String(),
				InstallHint:        "",
				ProvideClusterInfo: true,
			},
		},
	})

	config := &clientcmdv1.Config{
		CurrentContext: name,
		Clusters:       []clientcmdv1.NamedCluster{},
		Contexts:       []clientcmdv1.NamedContext{},
		AuthInfos:      authInfos,
	}

	extension := v1alpha1.ExecPluginConfig{
		ShootRef: v1alpha1.ShootRef{
			Namespace: k.namespace,
			Name:      k.shootName,
		},
		GardenClusterIdentity: k.gardenClusterIdentity,
	}

	var clusterExtensions []clientcmdv1.NamedExtension

	if !legacy {
		raw, err := json.Marshal(extension)
		if err != nil {
			return nil, fmt.Errorf("could not json marshal cluster extension: %w", err)
		}

		clusterExtensions = []clientcmdv1.NamedExtension{
			{
				Name: "client.authentication.k8s.io/exec",
				Extension: runtime.RawExtension{
					Raw: raw,
				},
			},
		}
	}

	for _, cluster := range k.clusters {
		name := fmt.Sprintf("%s-%s", authName, cluster.name)

		config.Clusters = append(config.Clusters, clientcmdv1.NamedCluster{
			Name: name,
			Cluster: clientcmdv1.Cluster{
				CertificateAuthorityData: cluster.caCert,
				Server:                   fmt.Sprintf("https://%s", cluster.apiServerHost),
				Extensions:               clusterExtensions,
			},
		})
		config.Contexts = append(config.Contexts, clientcmdv1.NamedContext{
			Name: name,
			Context: clientcmdv1.Context{
				Cluster:  name,
				AuthInfo: authName,
			},
		})
	}

	return runtime.Encode(clientcmdlatest.Codec, config)
}

// getConfig returns the util.ControllerManagerConfiguration of the ShootReconciler
func (r *ShootReconciler) getConfig() *util.ControllerManagerConfiguration {
	r.configMutex.RLock()
	defer r.configMutex.RUnlock()

	return r.Config
}

//// injectConfig is mainly used for tests to inject util.ControllerManagerConfiguration configuration
//func (r *ShootReconciler) injectConfig(config *util.ControllerManagerConfiguration) {
//	r.configMutex.Lock()
//	defer r.configMutex.Unlock()
//
//	r.Config = config
//}

func (r *ShootReconciler) increaseCounterForNamespace(namespace string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var counter int
	if c, exists := r.ReconcilerCountPerNamespace[namespace]; !exists {
		counter = 1
	} else {
		counter = c + 1
	}

	if counter > r.getConfig().Controllers.Shoot.MaxConcurrentReconcilesPerNamespace {
		return fmt.Errorf("max count reached")
	}

	r.ReconcilerCountPerNamespace[namespace] = counter

	return nil
}

func (r *ShootReconciler) decreaseCounterForNamespace(namespace string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var counter int

	c, exists := r.ReconcilerCountPerNamespace[namespace]
	if !exists {
		panic("entry expected!")
	}

	counter = c - 1
	if counter == 0 {
		delete(r.ReconcilerCountPerNamespace, namespace)
	} else {
		r.ReconcilerCountPerNamespace[namespace] = counter
	}
}
