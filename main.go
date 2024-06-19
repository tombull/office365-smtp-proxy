package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"

	"github.com/andrewheberle/graph-smtpd/pkg/graphserver"
	"github.com/cloudflare/certinel/fswatcher"
	"github.com/emersion/go-smtp"
	"github.com/oklog/run"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	// general options
	pflag.Bool("debug", false, "Enable debug mode")

	// SMTP options
	pflag.String("addr", "localhost:2525", "Service listen address")
	pflag.String("domain", "localhost", "Service domain/hostname")
	pflag.Int("recipients", 10, "Maximum message recipients")
	pflag.Int64("max", 1024*1024, "Maximum message size in bytes")

	// Access controls
	pflag.StringSlice("senders", []string{}, "List of allowed senders")
	pflag.StringSlice("sources", []string{}, "Source IP addresses allowed to relay")

	// TLS options
	pflag.String("cert", "", "TLS certificate")
	pflag.String("key", "", "TLS key")

	// Entra ID options
	pflag.String("clientid", "", "App Registration Client/Application ID")
	pflag.String("tenantid", "", "App Registration Tenant ID")
	pflag.String("secret", "", "App Registration Client Secret")
	pflag.Parse()

	// viper setup
	viper.SetEnvPrefix("smtpd")
	viper.AutomaticEnv()
	viper.BindPFlags(pflag.CommandLine)

	// set up backend
	var be *graphserver.Backend

	if viper.GetBool("debug") {
		b, err := graphserver.NewDebugGraphBackend(viper.GetString("clientid"), viper.GetString("tenantid"), viper.GetString("secret"))
		if err != nil {
			slog.Error("error setting up backend", "error", err)
			os.Exit(1)
		}
		be = b
	} else {
		b, err := graphserver.NewGraphBackend(viper.GetString("clientid"), viper.GetString("tenantid"), viper.GetString("secret"))
		if err != nil {
			slog.Error("error setting up backend", "error", err)
			os.Exit(1)
		}
		be = b
	}

	// add logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	be.SessionLog = logger

	// add access control
	be.SetAllowedSenders(viper.GetStringSlice("senders"))
	be.SetAllowedSources(viper.GetStringSlice("sources"))

	logger.Info("graph backend created")

	// set up server
	s := smtp.NewServer(be)
	s.Addr = viper.GetString("addr")
	s.Domain = viper.GetString("domain")
	s.MaxRecipients = viper.GetInt("recipients")
	s.MaxMessageBytes = viper.GetInt64("max")

	// set up run group
	g := run.Group{}

	if viper.GetString("cert") != "" && viper.GetString("key") != "" {
		ctx, cancel := context.WithCancel(context.Background())

		certinel, err := fswatcher.New(viper.GetString("cert"), viper.GetString("key"))
		if err != nil {
			logger.Error("could not set up certinel", "error", err, "cert", viper.GetString("cert"), "key", viper.GetString("key"))
			os.Exit(1)
		}

		// add certinel
		g.Add(func() error {
			logger.Info("starting up", "from", "certificate watcher", "cert", viper.GetString("cert"), "key", viper.GetString("key"))
			return certinel.Start(ctx)
		}, func(err error) {
			if err != nil {
				logger.Error("error on exit", "from", "certificate watcher", "error", err)
			}
			cancel()
		})

		// set up certiifcate watching for server
		s.TLSConfig = &tls.Config{
			GetCertificate: certinel.GetCertificate,
		}
	}

	// add SMTP server
	g.Add(func() error {
		logger.Info("starting up", "from", "SMTP server", "addr", viper.GetString("addr"), "domain", viper.GetString("domain"))
		return s.ListenAndServe()
	}, func(err error) {
		if err != nil {
			logger.Error("error on exit", "from", "SMTP server", "error", err)
		}
		s.Close()
	})

	logger.Info("starting components")

	if err := g.Run(); err != nil {
		logger.Error("run group error", "error", err)
		os.Exit(1)
	}
}
