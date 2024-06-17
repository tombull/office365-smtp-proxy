package main

import (
	"log"

	"github.com/andrewheberle/graph-smtpd/pkg/graphserver"
	"github.com/emersion/go-smtp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	// command line flags
	pflag.String("addr", "localhost:2525", "Service listen address")
	pflag.String("domain", "localhost", "Service domain/hostname")
	pflag.Bool("insecure", false, "Allow insecure authentication methods")
	pflag.Int("recipients", 10, "Maximum message recipients")
	pflag.Int64("max", 1024*1024, "Maximum message size in bytes")
	pflag.String("clientid", "", "App Registration Client/Application ID")
	pflag.String("tenantid", "", "App Registration Tenant ID")
	pflag.String("secret", "", "App Registration Client Secret")
	pflag.Parse()

	// viper setup
	viper.SetEnvPrefix("smtpd")
	viper.AutomaticEnv()
	viper.BindPFlags(pflag.CommandLine)

	// set up backend
	be := graphserver.NewGraphBackend(viper.GetString("clientid"), viper.GetString("tenantid"), viper.GetString("secret"))

	// set up server
	s := smtp.NewServer(be)
	s.Addr = viper.GetString("addr")
	s.Domain = viper.GetString("domain")
	s.AllowInsecureAuth = viper.GetBool("insecure")
	s.MaxRecipients = viper.GetInt("recipients")
	s.MaxMessageBytes = viper.GetInt64("max")
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
