package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/OfimaticSRL/parsemail"
	"github.com/andrewheberle/graph-smtpd/pkg/sendmail"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
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

	// create graph client
	cred, err := azidentity.NewClientSecretCredential(
		viper.GetString("tenantid"),
		viper.GetString("clientId"),
		viper.GetString("secret"),
		nil,
	)
	if err != nil {
		slog.Error("could not create a cred from a secret", "error", err)
		os.Exit(1)
	}

	client, err := graph.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
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

	// grab headers and content
	header := msg.Header
	subject := header.Get("Subject")
	from := msg.Sender.String()
	to := header.Get("To")
	cc := header.Get("Cc")
	bcc := header.Get("Bcc")

	// create the request ready to POST
	requestBody, err := sendmail.NewMessage(from, to, subject,
		sendmail.WithCc(cc),
		sendmail.WithBcc(bcc),
		sendmail.WithAttachments(msg.Attachments),
		sendmail.WithSaveToSentItems(viper.GetBool("sentitems")),
	).SendMailPostRequestBody()
	if err != nil {
		slog.Error("unable to create send email request", "error", err)
		os.Exit(1)
	}

	// send email
	if err := client.Users().ByUserId(from).SendMail().Post(context.Background(), requestBody, nil); err != nil {
		slog.Error("error sending email", "error", err)
		os.Exit(1)
	}
}
