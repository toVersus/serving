/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"log"

	"k8s.io/client-go/tools/clientcmd"

	"go.uber.org/zap"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/logging/logkey"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/system"
	"knative.dev/pkg/version"
	"knative.dev/pkg/webhook"
	kpa "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
	apiconfig "github.com/knative/serving/pkg/apis/config"
	net "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

const (
	component = "webhook"
)

var (
	masterURL  = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
)

func main() {
	flag.Parse()
	cm, err := configmap.Load("/etc/config-logging")
	if err != nil {
		log.Fatal("Error loading logging configuration:", err)
	}
	config, err := logging.NewConfigFromMap(cm)
	if err != nil {
		log.Fatal("Error parsing logging configuration:", err)
	}
	logger, atomicLevel := logging.NewLoggerFromConfig(config, component)
	defer logger.Sync()
	logger = logger.With(zap.String(logkey.ControllerType, component))

	logger.Info("Starting the Configuration Webhook")

	// Set up signals so we handle the first shutdown signal gracefully.
	stopCh := signals.SetupSignalHandler()

	clusterConfig, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		logger.Fatalw("Failed to get cluster config", zap.Error(err))
	}

	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		logger.Fatalw("Failed to get the client set", zap.Error(err))
	}

	if err := version.CheckMinimumVersion(kubeClient.Discovery()); err != nil {
		logger.Fatalw("Version check failed", err)
	}

	// Watch the logging config map and dynamically update logging levels.
	configMapWatcher := configmap.NewInformedWatcher(kubeClient, system.Namespace())
	configMapWatcher.Watch(logging.ConfigMapName(), logging.UpdateLevelFromConfigMap(logger, atomicLevel, component))

	store := apiconfig.NewStore(logger.Named("config-store"))
	store.WatchConfigs(configMapWatcher)

	if err = configMapWatcher.Start(stopCh); err != nil {
		logger.Fatalw("Failed to start the ConfigMap watcher", zap.Error(err))
	}

	options := webhook.ControllerOptions{
		ServiceName:    "webhook",
		DeploymentName: "webhook",
		Namespace:      system.Namespace(),
		Port:           8443,
		SecretName:     "webhook-certs",
		WebhookName:    "webhook.serving.knative.dev",
	}
	controller := webhook.AdmissionController{
		Client:  kubeClient,
		Options: options,
		Handlers: map[schema.GroupVersionKind]webhook.GenericCRD{
			v1alpha1.SchemeGroupVersion.WithKind("Revision"):      &v1alpha1.Revision{},
			v1alpha1.SchemeGroupVersion.WithKind("Configuration"): &v1alpha1.Configuration{},
			v1alpha1.SchemeGroupVersion.WithKind("Route"):         &v1alpha1.Route{},
			v1alpha1.SchemeGroupVersion.WithKind("Service"):       &v1alpha1.Service{},
			v1beta1.SchemeGroupVersion.WithKind("Revision"):       &v1beta1.Revision{},
			v1beta1.SchemeGroupVersion.WithKind("Configuration"):  &v1beta1.Configuration{},
			v1beta1.SchemeGroupVersion.WithKind("Route"):          &v1beta1.Route{},
			v1beta1.SchemeGroupVersion.WithKind("Service"):        &v1beta1.Service{},
			kpa.SchemeGroupVersion.WithKind("PodAutoscaler"):      &kpa.PodAutoscaler{},
			net.SchemeGroupVersion.WithKind("Certificate"):        &net.Certificate{},
			net.SchemeGroupVersion.WithKind("ClusterIngress"):     &net.ClusterIngress{},
			net.SchemeGroupVersion.WithKind("ServerlessService"):  &net.ServerlessService{},
		},
		Logger:                logger,
		DisallowUnknownFields: true,

		// Decorate contexts with the current state of the config.
		WithContext: func(ctx context.Context) context.Context {
			return v1beta1.WithUpgradeViaDefaulting(store.ToContext(ctx))
		},
	}
	if err = controller.Run(stopCh); err != nil {
		logger.Fatalw("Failed to start the admission controller", zap.Error(err))
	}
}
