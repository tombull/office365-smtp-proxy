package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/OfimaticSRL/parsemail"
	"github.com/andrewheberle/graph-smtpd/pkg/graphclient"
	"github.com/andrewheberle/graph-smtpd/pkg/sendmail"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	// general options
	pflag.String("config", "", "Configuration file")
	pflag.Bool("debug", false, "Enable debugging")

	// Entra ID options
	pflag.String("clientid", "", "App Registration Client/Application ID")
	pflag.String("tenantid", "", "App Registration Tenant ID")
	pflag.String("secret", "", "App Registration Client Secret")
	pflag.Parse()

	// sending options
	pflag.Bool("sentitems", false, "Save to sent items in senders mailbox")

	// viper setup
	viper.SetEnvPrefix("sendmail")
	viper.AutomaticEnv()
	viper.BindPFlags(pflag.CommandLine)

	// set up logger
	logLevel := new(slog.LevelVar)
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// discard time
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}

			return a
		},
	})
	slog.SetDefault(slog.New(h))
	if viper.GetBool("debug") {
		logLevel.Set(slog.LevelDebug)
	}

	// load config file
	config := viper.GetString("config")
	if config != "" {
		viper.SetConfigFile(config)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc/graph-sendmail")
		viper.AddConfigPath("$HOME/.graph-sendmail")
		viper.AddConfigPath(".")
	}
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if config != "" {
				slog.Error("config file not found", "error", err, "config", config)
				os.Exit(1)
			} else {
				slog.Info("running without config")
			}
		} else {
			slog.Error("config file was invalid", "error", err, "config", viper.ConfigFileUsed())
			os.Exit(1)
		}
	} else {
		slog.Info("config file loaded", "config", viper.ConfigFileUsed())
	}

	// create graph client
	client, err := graphclient.NewClient(viper.GetString("tenantid"), viper.GetString("clientId"), viper.GetString("secret"))
	if err != nil {
		slog.Error("could not create graph client", "error", err)
		os.Exit(1)
	}

	// parse incoming message
	msg, err := parsemail.Parse(os.Stdin)
	if err != nil {
		slog.Error("unable to read message", "error", err)
		os.Exit(1)
	}

	slog.Debug("message", "msg", msg)

	// grab headers and content
	header := msg.Header
	subject := header.Get("Subject")
	from := header.Get("From")
	to := header.Get("To")
	cc := header.Get("Cc")
	bcc := header.Get("Bcc")

	logger := slog.With("from", from, "to", to, "subject", subject)
	if cc != "" {
		logger = logger.With("cc", cc)
	}

	if bcc != "" {
		logger = logger.With("bcc", bcc)
	}

	opts := []sendmail.MessageOption{
		sendmail.WithCc(cc),
		sendmail.WithBcc(bcc),
		sendmail.WithAttachments(msg.Attachments),
		sendmail.WithSaveToSentItems(viper.GetBool("sentitems")),
	}

	// handle HTML or text bodies
	if msg.TextBody == "" {
		logger = logger.With("size", len(msg.HTMLBody), "type", "text/html")
		opts = append(opts, sendmail.WithBody(msg.HTMLBody), sendmail.WithHTMLContent())
	} else if msg.HTMLBody == "" {
		logger = logger.With("size", len(msg.TextBody), "type", "text/plain")
		opts = append(opts, sendmail.WithBody(msg.TextBody))
	}

	// create the request ready to POST
	requestBody, err := sendmail.NewMessage(from, to, subject, opts...).SendMailPostRequestBody()
	if err != nil {
		logger.Error("unable to create send email request", "error", err)
		os.Exit(1)
	}

	// send email
	if err := client.Users().ByUserId(from).SendMail().Post(context.Background(), requestBody, nil); err != nil {
		logger.Error("error sending email", "error", err)
		os.Exit(1)
	}

	logger.Info("message sent")
}
