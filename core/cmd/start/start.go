// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter/http"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/listener"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/console"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/driver"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/transport"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	xservice "github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type CoreConfig struct {
	insecure             *bool
	fips                 *bool
	debug                *bool
	autoCert             *bool
	validateChannelNames *bool
	memBacked            *bool
	noDriver             *bool
	updateBackupDisabled *bool
	alamos.Instrumentation
	dataPath             string
	secretsKeyFile       string
	licenseKey           string
	rootCredentials      auth.Credentials
	listeners            listener.Configs
	disabledIntegrations []string
	enabledIntegrations  []string
	certFactoryConfig    cert.FactoryConfig
	taskShutdownTimeout  time.Duration
	taskPollInterval     time.Duration
	taskOpTimeout        time.Duration
	slowConsumerTimeout  time.Duration
	sessionIdleTimeout   time.Duration
	taskWorkerCount      uint8
	secretsKey           secret.EncryptionKey
}

var _ config.Config[CoreConfig] = CoreConfig{}

var DefaultCoreConfig = CoreConfig{
	certFactoryConfig: cert.DefaultFactoryConfig,
	// a Core installed before the flag existed starts without it
	sessionIdleTimeout: auth.DefaultIdleTimeout,
}

// driverEnabled reports whether the embedded Driver starts with this Core.
func (c CoreConfig) driverEnabled() bool { return c.noDriver != nil && !*c.noDriver }

// DriverTrustAnchors returns the PEM certificates the embedded Driver verifies the
// Core against: the Core CA and the node certificate. It returns nil in insecure mode.
func DriverTrustAnchors(insecure bool, lc cert.LoaderConfig) ([]byte, error) {
	if insecure {
		return nil, nil
	}
	l, err := cert.NewLoader(lc)
	if err != nil {
		return nil, err
	}
	return l.TrustAnchorsPEM()
}

func (c CoreConfig) Validate() error {
	v := validate.New("core.config")
	v.NotNil("insecure", c.insecure)
	v.NotNil("fips", c.fips)
	v.NotNil("debug", c.debug)
	v.NotNil("auto_cert", c.autoCert)
	v.NotNil("mem_backed", c.memBacked)
	v.Exec(c.listeners.Validate)
	if c.insecure != nil && !*c.insecure && c.driverEnabled() {
		v.Exec(c.listeners.ValidateAdvertiseSource)
	}
	v.NotEmptyString("data_path", c.dataPath)
	v.NonZero("slow_consumer_timeout", c.slowConsumerTimeout)
	v.Positive("session_idle_timeout", c.sessionIdleTimeout)
	v.NotNil("no_driver", c.noDriver)
	v.NotNil("update_backup_disabled", c.updateBackupDisabled)
	v.Exec(c.rootCredentials.Validate)
	v.NonZero("task_op_timeout", c.taskOpTimeout)
	v.NonZero("task_poll_interval", c.taskPollInterval)
	v.NonZero("task_shutdown_timeout", c.taskShutdownTimeout)
	v.NonZero("task_worker_count", c.taskWorkerCount)
	v.NotNil("validate_channel_names", c.validateChannelNames)
	v.Exec(c.certFactoryConfig.Validate)
	return v.Error()
}

func (c CoreConfig) Override(other CoreConfig) CoreConfig {
	return CoreConfig{
		Instrumentation: override.Zero(c.Instrumentation, other.Instrumentation),
		insecure:        override.Nil(c.insecure, other.insecure),
		fips:            override.Nil(c.fips, other.fips),
		debug:           override.Nil(c.debug, other.debug),
		autoCert:        override.Nil(c.autoCert, other.autoCert),
		licenseKey:      override.String(c.licenseKey, other.licenseKey),
		memBacked:       override.Nil(c.memBacked, other.memBacked),
		listeners:       override.Slice(c.listeners, other.listeners),
		dataPath:        override.String(c.dataPath, other.dataPath),
		secretsKey: override.If(
			c.secretsKey,
			other.secretsKey,
			other.secretsKey != secret.EncryptionKey{},
		),
		secretsKeyFile: override.String(c.secretsKeyFile, other.secretsKeyFile),
		slowConsumerTimeout: override.Numeric(
			c.slowConsumerTimeout,
			other.slowConsumerTimeout,
		),
		rootCredentials: override.Zero(c.rootCredentials, other.rootCredentials),
		sessionIdleTimeout: override.Numeric(
			c.sessionIdleTimeout,
			other.sessionIdleTimeout,
		),
		noDriver: override.Nil(c.noDriver, other.noDriver),
		updateBackupDisabled: override.Nil(
			c.updateBackupDisabled,
			other.updateBackupDisabled,
		),
		taskOpTimeout: override.Numeric(c.taskOpTimeout, other.taskOpTimeout),
		taskPollInterval: override.Numeric(
			c.taskPollInterval,
			other.taskPollInterval,
		),
		taskShutdownTimeout: override.Numeric(
			c.taskShutdownTimeout,
			other.taskShutdownTimeout,
		),
		taskWorkerCount: override.Numeric(
			c.taskWorkerCount,
			other.taskWorkerCount,
		),
		certFactoryConfig: c.certFactoryConfig.Override(other.certFactoryConfig),
		enabledIntegrations: override.Slice(
			c.enabledIntegrations,
			other.enabledIntegrations,
		),
		disabledIntegrations: override.Slice(
			c.disabledIntegrations,
			other.disabledIntegrations,
		),
		validateChannelNames: override.Nil(
			c.validateChannelNames,
			other.validateChannelNames,
		),
	}
}

const secretsKeyFileName = "secrets.key"

// resolveSecretsKey returns the key that encrypts stored secrets: the key setting, the
// key file setting, a random key for a Core in memory, or the key file in the data
// directory. A missing key file gets a new random key.
func (c CoreConfig) resolveSecretsKey() (secret.EncryptionKey, error) {
	if c.secretsKey != (secret.EncryptionKey{}) {
		return c.secretsKey, nil
	}
	if c.secretsKeyFile != "" {
		return secret.LoadEncryptionKey(c.secretsKeyFile)
	}
	if *c.memBacked {
		return secret.NewEncryptionKey(), nil
	}
	return secret.LoadEncryptionKey(filepath.Join(c.dataPath, secretsKeyFileName))
}

// BootupCore contains the most important Core startup logic. It does and should not
// read any variables from viper, and instead should be called with  fully configured
// CoreConfigs.
func BootupCore(
	ctx context.Context,
	onServerStarted chan struct{},
	cfgs ...CoreConfig,
) (err error) {
	cfg, err := config.New(DefaultCoreConfig, cfgs...)
	if err != nil {
		return err
	}

	if *cfg.autoCert {
		if err = cmdcert.GenerateAuto(cfg.certFactoryConfig); err != nil {
			return errors.Wrap(err, "failed to generate auto certs")
		}
	}

	vsn := version.Get()

	cfg.L.Zap().Sugar().Infof("\033[34mSynnax version %s starting\033[0m", vsn)
	cfg.L.Info(
		"starting synnax node",
		zap.String("version", vsn),
		zap.String("commit", version.Commit()),
		zap.Time("build", version.Time()),
	)

	// Any data stored on the node is considered sensitive, so we need to set the
	// permission mask for all files appropriately.
	DisablePermissionBits()

	var (
		closer xio.MultiCloser
		// update is the update of the data that this start did not commit yet.
		update *storage.Update
	)
	cleanup, ok := xservice.NewOpener(ctx, &closer)
	defer func() {
		if err = cleanup(err); err != nil && update != nil {
			err = update.Undo(ctx, err)
		}
	}()

	securityProvider, err := security.NewProvider(security.ProviderConfig{
		LoaderConfig: cfg.certFactoryConfig.LoaderConfig,
		Insecure:     cfg.insecure,
	})
	if !ok(err, nil) {
		return errors.Mark(err, validate.ErrValidation)
	}

	cacheDir, err := os.UserCacheDir()
	if !ok(err, nil) {
		return errors.Wrap(err, "failed to resolve cache directory")
	}
	cacheDir = filepath.Join(cacheDir, "synnax", "core")
	workDir, closeWorkDir, err := openWorkDir(cacheDir)
	if !ok(err, closeWorkDir) {
		return errors.Wrapf(err, "failed to resolve working directory")
	}
	cfg.L.Info("using working directory", zap.String("dir", workDir))

	storageLayer, err := storage.OpenLayer(ctx, storage.LayerConfig{
		Instrumentation:      cfg.Child("storage"),
		InMemory:             cfg.memBacked,
		Dirname:              cfg.dataPath,
		Version:              vsn,
		UpdateBackupDisabled: cfg.updateBackupDisabled,
	})
	if !ok(err, storageLayer) {
		return err
	}
	update = storageLayer.Update

	distributionLayer, err := distribution.OpenLayer(ctx, distribution.LayerConfig{
		Instrumentation:  cfg.Child("distribution"),
		AdvertiseAddress: cfg.listeners.AdvertiseAddress(),
		Storage:          storageLayer,
	})
	if !ok(err, distributionLayer) {
		return err
	}

	secretsKey, err := cfg.resolveSecretsKey()
	if !ok(err, nil) {
		return err
	}

	serviceLayer, err := service.OpenLayer(ctx, service.LayerConfig{
		Instrumentation:    cfg.Child("service"),
		Distribution:       distributionLayer,
		Security:           securityProvider,
		Storage:            storageLayer,
		RootCredentials:    cfg.rootCredentials,
		FIPS:               cfg.fips,
		SessionIdleTimeout: cfg.sessionIdleTimeout,
		License: license.ServiceConfig{
			Key:      cfg.licenseKey,
			Version:  version.Get(),
			Loopback: new(cfg.listeners.Loopback()),
		},
		ValidateChannelNames: cfg.validateChannelNames,
		SecretsKey:           secretsKey,
	})
	if !ok(err, serviceLayer) {
		return err
	}

	// No client reaches the data before the commit, so an undo loses no writes.
	if update != nil {
		if err = serviceLayer.License.CheckUpdate(update.From); err != nil {
			return err
		}
		if err = update.Commit(); err != nil {
			return err
		}
		cfg.L.Info(
			"committed the update",
			zap.String("from", update.From),
			zap.String("to", update.To),
		)
		update = nil
	}

	apiLayer, err := api.NewLayer(api.LayerConfig{
		Instrumentation: cfg.Child("api"),
		Service:         serviceLayer,
		Distribution:    distributionLayer,
	})
	if !ok(err, nil) {
		return err
	}

	var r *http.Router
	if r, err = http.NewRouter(http.RouterConfig{
		Instrumentation:     cfg.Instrumentation,
		StreamWriteDeadline: cfg.slowConsumerTimeout,
	}); !ok(err, nil) {
		return err
	}
	transportLayer, err := transport.NewLayer(transport.LayerConfig{
		Instrumentation: cfg.Child("transport"),
		API:             apiLayer,
		Router:          r,
	})
	if !ok(err, nil) {
		return err
	}

	var embeddedConsole *console.Console
	if embeddedConsole, err = console.New(
		console.Config{Insecure: cfg.insecure},
	); !ok(err, nil) {
		return err
	}

	serverListeners, err := cfg.listeners.Resolve(
		securityProvider,
		cfg.certFactoryConfig,
		*cfg.insecure,
		cfg.driverEnabled(),
	)
	if !ok(err, nil) {
		return err
	}

	if rootServer, err := server.Serve(
		server.Config{
			Branches: []server.Branch{
				&server.SecureHTTPBranch{
					Transports: []http.BindableTransport{r, embeddedConsole},
				},
				&server.GRPCBranch{Transports: transportLayer.GRPC},
				server.NewHTTPRedirectBranch(),
			},
			Debug:           cfg.debug,
			Listeners:       serverListeners,
			Instrumentation: cfg.Child("server"),
			Security:        server.SecurityConfig{Insecure: cfg.insecure},
		},
	); !ok(err, rootServer) {
		return err
	}

	// The embedded Driver dials the advertised listener, which does not always serve
	// the node certificate.
	driverTrustAnchors, err := DriverTrustAnchors(
		*cfg.insecure,
		cfg.certFactoryConfig.LoaderConfig,
	)
	if !ok(err, nil) {
		return err
	}

	licenseInfo := serviceLayer.License.Retrieve()
	covered := licenseInfo.State == license.StateOk
	if embeddedDriver, err := driver.Open(
		ctx,
		driver.Config{
			Enabled:  new(!*cfg.noDriver),
			Insecure: cfg.insecure,
			// An unlicensed Core refuses the Driver's rack, so the Driver retries in
			// the background.
			Detached: new(!covered),
			Integrations: parseIntegrations(
				cfg.enabledIntegrations,
				cfg.disabledIntegrations,
			),
			Instrumentation:     cfg.Child("driver"),
			Address:             cfg.listeners.AdvertiseAddress(),
			RackKey:             serviceLayer.Rack.EmbeddedKey,
			Keys:                serviceLayer.APIKey,
			DB:                  distributionLayer.DB,
			KeyDirname:          lo.Ternary(*cfg.memBacked, "", cfg.dataPath),
			Debug:               cfg.debug,
			TrustAnchorsPEM:     driverTrustAnchors,
			ParentDirname:       workDir,
			BinaryDirname:       filepath.Join(cacheDir, "driver"),
			TaskWorkerCount:     cfg.taskWorkerCount,
			TaskShutdownTimeout: cfg.taskShutdownTimeout,
			TaskPollInterval:    cfg.taskPollInterval,
			TaskOpTimeout:       cfg.taskOpTimeout,
		},
	); !ok(err, embeddedDriver) {
		return err
	}

	cfg.L.Infof(
		"\033[32mSynnax is running and available at %v \033[0m",
		cfg.listeners.AdvertiseAddress(),
	)
	if !covered {
		scheme := "https"
		if *cfg.insecure {
			scheme = "http"
		}
		console := scheme + "://" + string(cfg.listeners.AdvertiseAddress())
		if len(licenseInfo.Fingerprint) == 0 {
			cfg.L.Warn(fmt.Sprintf(
				"no active license on this Core, and it reports no network hardware "+
					"to bind one to. Ask Synnax Labs for a floating license, then "+
					"activate it in the Console at %s.",
				console,
			))
		} else {
			hostname, err := os.Hostname()
			if err != nil {
				cfg.L.Warn(
					"failed to read the host name for the activation link",
					zap.Error(err),
				)
			}
			cfg.L.Warn(fmt.Sprintf(
				"no active license on this Core. To activate it, open the Console at "+
					"%s, or open this link on any computer: %s",
				console,
				licenseInfo.Fingerprint.ActivationURL(hostname),
			))
		}
	}

	if onServerStarted != nil {
		onServerStarted <- struct{}{}
	}
	<-ctx.Done()
	return err
}

func openWorkDir(cacheDir string) (string, io.Closer, error) {
	dir := filepath.Join(cacheDir, "workdir", strconv.Itoa(os.Getpid()))
	if err := os.MkdirAll(dir, fs.UserRWX); err != nil {
		return "", nil, err
	}
	return dir, xio.CloserFunc(func() error { return os.RemoveAll(dir) }), nil
}

func parseIntegrations(enabled, disabled []string) []string {
	if len(enabled) > 0 {
		return enabled
	}
	return lo.Filter(driver.AllIntegrations, func(integration string, _ int) bool {
		return !lo.Contains(disabled, integration)
	})
}
