/*
Copyright 2024.

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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	albv1alpha1 "github.com/yandex-cloud/yc-alb-ingress-controller/api/v1alpha1"
	"github.com/yandex-cloud/yc-alb-ingress-controller/controllers/grpcbackendgroup"
	"github.com/yandex-cloud/yc-alb-ingress-controller/controllers/httpbackendgroup"
	"github.com/yandex-cloud/yc-alb-ingress-controller/controllers/ingress"
	"github.com/yandex-cloud/yc-alb-ingress-controller/controllers/secret"
	"github.com/yandex-cloud/yc-alb-ingress-controller/controllers/service"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/builders"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/deploy"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/k8s"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/metadata"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/reconcile"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/yc"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(albv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// this var is supposed to be injected in docker build
var userAgent = "alb-ingress-controller"

func main() {
	var (
		probeAddr         string
		useEndpointSlices bool
	)

	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&useEndpointSlices, "use-endpoint-slices", false,
		"Use newer endpoint slices API instead of endpoints. "+
			"Does not affect behavior, but will be used by default when endpoints api is deprecated")

	var (
		folderID                  string
		certsFolderID             string
		clusterID                 string
		region                    string
		clusterLabelName          string
		keyFile                   string
		endpoint                  string
		enableDefaultHealthChecks bool
	)
	flag.StringVar(&folderID, "folder-id", "", "alb folder ID")
	flag.StringVar(&certsFolderID, "certs-folder-id", "", "certificates folder ID, by default equals to value of folder-id")
	flag.StringVar(&clusterID, "cluster-id", "", "arbitrary cluster identifier")
	flag.StringVar(&region, "region", "", "region")
	flag.StringVar(&clusterLabelName, "cluster-label-name", "cluster_ref_label", "common label for cloud resources for ingress controller")
	flag.StringVar(&keyFile, "keyfile", "", "service account key json file")
	flag.StringVar(&endpoint, "endpoint", "", "cloud environment endpoint (defaults to prod endpoint)")
	flag.BoolVar(&enableDefaultHealthChecks, "enable-default-health-checks", true, "enables default healthchecks in ALB configuration")

	opts := zap.Options{
		Development:     true,
		StacktraceLevel: zapcore.DPanicLevel,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if clusterID == "" {
		clusterID = os.Getenv("YC_ALB_CLUSTER_ID")
	}
	if clusterID == "" {
		clusterID = "default"
	}

	if folderID == "" {
		folderID = os.Getenv("YC_ALB_FOLDER_ID")
	}
	if folderID == "" {
		setupLog.Error(fmt.Errorf("folder-id missing"), "")
		os.Exit(1)
	}

	if certsFolderID == "" {
		certsFolderID = folderID
	}

	if region == "" {
		region = os.Getenv("YC_ALB_REGION")
	}

	if endpoint == "" {
		endpoint = os.Getenv("YC_ENDPOINT")
	}

	if envEnable := os.Getenv("YC_ALB_ENABLE_DEFAULT_HEALTHCHECKS"); envEnable != "" {
		var err error
		enableDefaultHealthChecks, err = strconv.ParseBool(envEnable)
		if err != nil {
			setupLog.Error(err, "unable to parse YC_ALB_ENABLE_DEFAULT_HEALTHCHECKS")
			os.Exit(1)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		// Explicitly disable LeaderElection
		LeaderElection: false,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	sdk, err := buildSDK(keyFile, endpoint)
	if err != nil {
		setupLog.Error(err, "failed to build ycsdk")
		os.Exit(1)
	}
	setupLog.Info("sdk created")

	clientSet, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to obtain clientSet")
		os.Exit(1)
	}

	names := &metadata.Names{ClusterID: clusterID}
	labels := &metadata.Labels{
		ClusterLabelName: clusterLabelName,
		ClusterID:        clusterID,
	}

	cli := mgr.GetClient()
	repo := yc.NewRepository(sdk, names, folderID)
	builders.SetupDefaultHealthChecks(enableDefaultHealthChecks)
	resolvers := builders.NewResolvers(repo)

	if err = (&service.Reconciler{
		Repo: repo,

		TargetGroupBuilder:  reconcile.NewTargetGroupBuilder(folderID, cli, names, labels, repo.FindInstanceByID, useEndpointSlices),
		TargetGroupDeployer: deploy.NewServiceDeployer(repo),

		BackendGroupBuilder:  &builders.BackendGroupForSvcBuilder{FolderID: folderID, Names: names},
		BackendGroupDeployer: deploy.NewBackendGroupDeployer(repo),

		FinalizerManager:   &k8s.FinalizerManager{Client: cli},
		GroupStatusManager: k8s.NewGroupStatusManager(cli),
		ServiceLoader:      &k8s.DefaultServiceLoader{Client: cli},
		IngressLoader:      k8s.NewIngressLoader(cli),
		Names:              names,
		Resolvers:          resolvers,
	}).SetupWithManager(mgr, useEndpointSlices); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Service")
		os.Exit(1)
	}

	newEngineFn := func(d *builders.Data) *reconcile.IngressGroupEngine {
		return &reconcile.IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: &yc.UpdatePredicates{},
			Names:      names,
		}
	}
	factory := builders.NewFactory(folderID, region, names, labels, cli, repo)

	secretEventChan := make(chan event.GenericEvent)
	certRepo := yc.NewCertRepo(sdk, certsFolderID)

	if err = (&ingress.GroupReconciler{
		Loader:             k8s.NewGroupLoader(cli),
		Builder:            reconcile.NewDefaultDataBuilder(factory, resolvers, newEngineFn, folderID, names, certRepo, repo),
		Deployer:           deploy.NewIngressGroupDeployManager(repo),
		StatusUpdater:      &k8s.StatusUpdater{Client: cli},
		FinalizerManager:   &k8s.FinalizerManager{Client: cli},
		GroupStatusManager: k8s.NewGroupStatusManager(cli),
		StatusResolver:     &reconcile.IngressStatusResolver{},
		SettingsLoader:     &k8s.GroupSettingsLoader{Client: cli},
		Scheme:             mgr.GetScheme(),
	}).SetupWithManager(mgr, clientSet, secretEventChan); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Ingress-Groups")
		os.Exit(1)
	}

	if err = (secret.NewController(cli, certRepo, names)).SetupWithManager(mgr, secretEventChan); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secrets")
		os.Exit(1)
	}

	httpBGRecHandler := &reconcile.HttpBackendGroupReconcileHandler{
		Repo:             repo,
		Predicates:       &yc.UpdatePredicates{},
		FinalizerManager: &k8s.FinalizerManager{Client: cli},

		Builder: &builders.HttpBackendGroupForCrdBuilder{
			FolderID: folderID,
			Names:    names,
			Cli:      cli,
			Repo:     repo,
		},
		Deployer: deploy.NewBackendGroupDeployer(repo),

		Names: names,
	}
	if err = (&httpbackendgroup.Reconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ReconcileHandler: httpBGRecHandler,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HttpBackendGroup")
		os.Exit(1)
	}

	grpcBGRecHandler := &reconcile.GrpcBackendGroupReconcileHandler{
		Repo:             repo,
		Predicates:       &yc.UpdatePredicates{},
		FinalizerManager: &k8s.FinalizerManager{Client: cli},

		Builder: &builders.GrpcBackendGroupForCrdBuilder{
			FolderID: folderID,
			Names:    names,
			Cli:      cli,
			Repo:     repo,
		},
		Deployer: deploy.NewBackendGroupDeployer(repo),

		Names: names,
	}
	if err = (&grpcbackendgroup.Reconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ReconcileHandler: grpcBGRecHandler,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HttpBackendGroup")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	setupLog.Info(fmt.Sprintf("Hello from %s!", userAgent))
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func buildSDK(keyFile, endpoint string) (*ycsdk.SDK, error) {
	var creds ycsdk.Credentials
	if len(keyFile) != 0 {
		key, err := getCredsFromFile(keyFile)
		if err != nil {
			return nil, err
		}
		creds, err = ycsdk.ServiceAccountKey(key)
		if err != nil {
			return nil, err
		}
		setupLog.Info("obtained credentials from keyfile")
	} else if token := os.Getenv("INGRESS_TOKEN"); token != "" {
		creds = ycsdk.NewIAMTokenCredentials(token)
		setupLog.Info("obtained credentials via token from environment variable")
	} else {
		return nil, fmt.Errorf("neither --keyfile flag nor INGRESS_TOKEN var has been provided")
	}
	return ycsdk.Build(context.Background(), ycsdk.Config{
		Credentials: creds,
		Endpoint:    endpoint,
	}, grpc.WithUserAgent(userAgent))
}

type Key struct {
	ID               string `json:"id"`
	PrivateKey       string `json:"private_key"`
	ServiceAccountID string `json:"service_account_id"`
}

func getCredsFromFile(keyFile string) (*iamkey.Key, error) {
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	key := &Key{}
	err = json.Unmarshal(data, key)
	if err != nil {
		return nil, err
	}
	return &iamkey.Key{
		Id:         key.ID,
		Subject:    &iamkey.Key_ServiceAccountId{ServiceAccountId: key.ServiceAccountID},
		PrivateKey: key.PrivateKey,
	}, nil
}
