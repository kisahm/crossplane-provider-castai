/*
Copyright 2021 Upbound Inc.
*/

package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/certificates"
	xpcontroller "github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/feature"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	tjcontroller "github.com/crossplane/upjet/pkg/controller"
	"github.com/crossplane/upjet/pkg/terraform"
	"gopkg.in/alecthomas/kingpin.v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane-contrib/crossplane-provider-castai/apis"
	"github.com/crossplane-contrib/crossplane-provider-castai/apis/v1alpha1"
	"github.com/crossplane-contrib/crossplane-provider-castai/config"
	"github.com/crossplane-contrib/crossplane-provider-castai/internal/clients"
	"github.com/crossplane-contrib/crossplane-provider-castai/internal/controller"
	"github.com/crossplane-contrib/crossplane-provider-castai/internal/features"
)

func main() {
	var (
		app                = kingpin.New(filepath.Base(os.Args[0]), "Terraform based Crossplane provider for CastAI").DefaultEnvars()
		debug              = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncInterval       = app.Flag("sync", "Sync interval controls how often all resources will be double checked for drift.").Short('s').Default("1h").Duration()
		pollInterval       = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").Default("10m").Duration()
		leaderElection     = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		terraformVersion   = app.Flag("terraform-version", "Terraform version.").Required().Envar("TERRAFORM_VERSION").String()
		providerSource     = app.Flag("terraform-provider-source", "Terraform provider source.").Required().Envar("TERRAFORM_PROVIDER_SOURCE").String()
		providerVersion    = app.Flag("terraform-provider-version", "Terraform provider version.").Required().Envar("TERRAFORM_PROVIDER_VERSION").String()
		maxReconcileRate   = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()
		nativeProviderPath = app.Flag("terraform-native-provider-path", "Terraform native provider path for shared execution.").Default("").Envar("TERRAFORM_NATIVE_PROVIDER_PATH").String()

		namespace                  = app.Flag("namespace", "Namespace used to set as default scope in default secret store config.").Default("upbound-system").Envar("POD_NAMESPACE").String()
		essTLSCertsPath            = app.Flag("ess-tls-cert-dir", "Path of ESS TLS certificates.").Envar("ESS_TLS_CERTS_DIR").String()
		enableExternalSecretStores = app.Flag("enable-external-secret-stores", "Enable support for ExternalSecretStores.").Default("false").Envar("ENABLE_EXTERNAL_SECRET_STORES").Bool()
		enableManagementPolicies   = app.Flag("enable-management-policies", "Enable support for Management Policies.").Default("true").Envar("ENABLE_MANAGEMENT_POLICIES").Bool()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("crossplane-provider-castai"))
	if *debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	}

	// currently, we configure the jitter to be the 5% of the poll interval
	pollJitter := time.Duration(float64(*pollInterval) * 0.05)
	log.Debug("Starting", "sync-interval", syncInterval.String(),
		"poll-interval", pollInterval.String(), "poll-jitter", pollJitter, "max-reconcile-rate", *maxReconcileRate)

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-crossplane-provider-castai",
		Cache: cache.Options{
			SyncPeriod: syncInterval,
		},
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")
	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add CastAI APIs to scheme")

	o := tjcontroller.Options{
		Options: xpcontroller.Options{
			Logger:                  log,
			GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
			PollInterval:            1 * time.Minute,
			MaxConcurrentReconciles: 1,
			Features:                &feature.Flags{},
		},
		Provider:   config.GetProvider(),
		SetupFn:    clients.TerraformSetupBuilder(*terraformVersion, *providerSource, *providerVersion),
		PollJitter: pollJitter,
	}

	if *enableManagementPolicies {
		o.Features.Enable(features.EnableBetaManagementPolicies)
		log.Info("Beta feature enabled", "flag", features.EnableBetaManagementPolicies)
	}

	o.WorkspaceStore = terraform.NewWorkspaceStore(log, terraform.WithDisableInit(len(*nativeProviderPath) != 0), terraform.WithProcessReportInterval(*pollInterval), terraform.WithFeatures(o.Features))

	if *enableExternalSecretStores {
		o.SecretStoreConfigGVK = &v1alpha1.StoreConfigGroupVersionKind
		log.Info("Alpha feature enabled", "flag", features.EnableAlphaExternalSecretStores)

		o.ESSOptions = &tjcontroller.ESSOptions{}
		if *essTLSCertsPath != "" {
			log.Info("ESS TLS certificates path is set. Loading mTLS configuration.")
			tCfg, err := certificates.LoadMTLSConfig(filepath.Join(*essTLSCertsPath, "ca.crt"), filepath.Join(*essTLSCertsPath, "tls.crt"), filepath.Join(*essTLSCertsPath, "tls.key"), false)
			kingpin.FatalIfError(err, "Cannot load ESS TLS config.")

			o.ESSOptions.TLSConfig = tCfg
		}

		// Ensure default store config exists.
		kingpin.FatalIfError(resource.Ignore(kerrors.IsAlreadyExists, mgr.GetClient().Create(context.Background(), &v1alpha1.StoreConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
			Spec: v1alpha1.StoreConfigSpec{
				// NOTE(turkenh): We only set required spec and expect optional
				// ones to properly be initialized with CRD level default values.
				SecretStoreConfig: xpv1.SecretStoreConfig{
					DefaultScope: *namespace,
				},
			},
		})), "cannot create default store config")
	}

	kingpin.FatalIfError(controller.Setup(mgr, o), "Cannot setup CastAI controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
