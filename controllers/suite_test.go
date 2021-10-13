/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/gardener/gardenlogin-controller-manager/internal/test"
	"github.com/gardener/gardenlogin-controller-manager/internal/util"
	"github.com/gardener/gardenlogin-controller-manager/webhooks"

	gardenenvtest "github.com/gardener/gardener/pkg/envtest"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	randomLength = 5
	charset      = "abcdefghijklmnopqrstuvwxyz0123456789"
)

var (
	k8sClient       client.Client
	testEnv         *gardenenvtest.GardenerTestEnvironment
	ctx             context.Context
	cancel          context.CancelFunc
	k8sManager      ctrl.Manager
	cmConfig        *util.ControllerManagerConfiguration
	validator       *webhooks.ConfigmapValidator
	shootReconciler *ShootReconciler
)

// TODO rename file to controllers_suite_test.go
func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(30 * time.Second)
	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	cmConfig = test.DefaultConfiguration()

	validator = &webhooks.ConfigmapValidator{
		Log:    ctrl.Log.WithName("webhooks").WithName("ConfigmapValidation"),
		Config: cmConfig,
	}

	environment := test.New(validator)
	testEnv = environment.GardenEnv
	k8sManager = environment.K8sManager
	k8sClient = environment.K8sClient

	shootReconciler = &ShootReconciler{
		Client:                      k8sManager.GetClient(),
		Log:                         ctrl.Log.WithName("controllers").WithName("Shoot"),
		Scheme:                      k8sManager.GetScheme(),
		Config:                      cmConfig,
		ReconcilerCountPerNamespace: map[string]int{},
	}
	err := shootReconciler.SetupWithManager(ctx, k8sManager, cmConfig.Controllers.Shoot)
	Expect(err).ToNot(HaveOccurred())

	environment.Start()
}, 60)

var _ = AfterSuite(func() {
	cancel()
	By("running cleanup actions")
	framework.RunCleanupActions()

	By("tearing down the test environment")
	Expect(testEnv.Stop()).To(Succeed())
}, 60)
