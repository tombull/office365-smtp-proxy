package graphserver

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/OfimaticSRL/parsemail"
	"github.com/emersion/go-smtp"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

type Session struct {
	from   string
	user   *users.UserItemRequestBuilder
	client *graph.GraphServiceClient
	debug  bool
}

type Message struct {
	From    string   `json:"from"`
	To      []string `json:"toRecipients"`
	Cc      []string `json:"ccRecipients"`
	Bcc     []string `json:"bccRecipients"`
	Subject string   `json:"subject"`
	Sender  string   `json:"sender"`
	Date    string   `json:"sentDateTime"`
	Body    string   `json:"body"`
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	slog.Info("MAIL FROM", "from", from)

	if s.client == nil {
		return fmt.Errorf("graph client not initialised")
	}

	// get UserItemRequestBuilder
	if user := s.client.Users().ByUserId(from); user != nil {
		s.user = user
		return nil
	}

	s.from = from

	// see if user exists
	return fmt.Errorf("user not found")
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	slog.Info("RCPT TO", "to", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	slog.Info("DATA")

	// parse incoming message
	msg, err := parsemail.Parse(r)
	if err != nil {
		return err
	}

	// grab headers and content
	header := msg.Header
	subject := header.Get("Subject")
	from := s.from
	to := header.Get("To")
	cc := header.Get("Cc")
	bcc := header.Get("Bcc")
	body := models.NewItemBody()
	body.SetContent(&msg.TextBody)

	// build the message
	message := models.NewMessage()
	message.SetBody(body)
	message.SetSubject(&subject)

	if addrs := parseAddressList(to); len(addrs) > 0 {
		message.SetToRecipients(addrs)
	}

	if addrs := parseAddressList(cc); len(addrs) > 0 {
		message.SetCcRecipients(addrs)
	}

	if addrs := parseAddressList(bcc); len(addrs) > 0 {
		message.SetBccRecipients(addrs)
	}

	// add sender/from
	recipient := models.NewRecipient()
	emailAddress := models.NewEmailAddress()
	emailAddress.SetAddress(&from)
	recipient.SetEmailAddress(emailAddress)
	message.SetFrom(recipient)

	// handle any attachments
	attachments := []models.Attachmentable{}
	for _, a := range msg.Attachments {
		data, err := io.ReadAll(a.Data)
		if err != nil {
			return err
		}
		attachment := models.NewFileAttachment()
		attachment.SetName(&a.Filename)
		attachment.SetContentType(&a.ContentType)
		attachment.SetContentBytes(data)

		// add to attachmentsList
		attachments = append(attachments, attachment)
	}

	// add if any attachments
	if len(attachments) > 0 {
		message.SetAttachments(attachments)
	}

	// create sendMail request
	requestBody := users.NewItemSendmailSendMailPostRequestBody()
	requestBody.SetMessage(message)

	// send it
	return s.user.SendMail().Post(context.Background(), requestBody, nil)
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

func parseAddressList(addresses string) []models.Recipientable {
	recipientList := []models.Recipientable{}

	if addresses == "" {
		return recipientList
	}

	// Split the address list by commas and trim spaces
	list := strings.Split(addresses, ",")
	for i := range list {
		address := strings.TrimSpace(list[i])

		// build recipient
		recipient := models.NewRecipient()
		emailAddress := models.NewEmailAddress()
		emailAddress.SetAddress(&address)
		recipient.SetEmailAddress(emailAddress)

		// add to list
		recipientList = append(recipientList, recipient)
	}

	return recipientList
}
