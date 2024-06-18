package main

import (
	"context"
	"crypto/tls"
	"log"
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
	pflag.Bool("insecure", false, "Allow insecure authentication methods")
	pflag.Int("recipients", 10, "Maximum message recipients")
	pflag.Int64("max", 1024*1024, "Maximum message size in bytes")

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

	slog.Info("graph backend created")

	// set up server
	s := smtp.NewServer(be)
	s.Addr = viper.GetString("addr")
	s.Domain = viper.GetString("domain")
	s.AllowInsecureAuth = viper.GetBool("insecure")
	s.MaxRecipients = viper.GetInt("recipients")
	s.MaxMessageBytes = viper.GetInt64("max")

	// set up run group
	g := run.Group{}

	if viper.GetString("cert") != "" && viper.GetString("key") != "" {
		ctx, cancel := context.WithCancel(context.Background())

		certinel, err := fswatcher.New(viper.GetString("cert"), viper.GetString("key"))
		if err != nil {
			log.Fatal(err)
		}

		// add certinel
		g.Add(func() error {
			return certinel.Start(ctx)
		}, func(err error) {
			cancel()
		})

		// set up certiifcate watching for server
		s.TLSConfig = &tls.Config{
			GetCertificate: certinel.GetCertificate,
		}

		// allow insecure auth always via TLS
		s.AllowInsecureAuth = true

		// add TLS enabled server
		g.Add(func() error {
			return s.ListenAndServeTLS()
		}, func(err error) {
			s.Close()
		})
	} else {
		// add non TLS enabled server
		g.Add(func() error {
			return s.ListenAndServe()
		}, func(err error) {
			s.Close()
		})
	}

	slog.Info("starting up")

	if err := g.Run(); err != nil {
		log.Fatal(err)
	}
}
