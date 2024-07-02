package graphserver

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"

	"github.com/OfimaticSRL/parsemail"
	"github.com/andrewheberle/graph-smtpd/pkg/sendmail"
	"github.com/emersion/go-smtp"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

type Session struct {
	from            string
	user            *users.UserItemRequestBuilder
	client          *graph.GraphServiceClient
	debug           bool
	saveToSentItems bool
	logger          *slog.Logger
	logLevel        slog.Level
	allowedSenders  []string
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	if s.logger != nil {
		s.logger = s.logger.With("from", from)
	}

	if s.client == nil {
		if s.logger != nil {
			s.logger = s.logger.With("mailerror", fmt.Errorf("graph client not initialised"))
		}
		s.logLevel = slog.LevelError
		return fmt.Errorf("graph client not initialised")
	}

	// check that sender is allowed
	if len(s.allowedSenders) > 0 {
		if _, found := slices.BinarySearch(s.allowedSenders, from); !found {
			if s.logger != nil {
				s.logger = s.logger.With("mailerror", fmt.Errorf("sender not allowed"))
			}
			s.logLevel = slog.LevelError
			return fmt.Errorf("sender not allowed")
		}
	}

	// get UserItemRequestBuilder
	if user := s.client.Users().ByUserId(from); user != nil {
		s.user = user
		return nil
	}

	s.from = from

	// Some error creating user object
	if s.logger != nil {
		s.logger = s.logger.With("mailerror", fmt.Errorf("user not found"))
	}
	s.logLevel = slog.LevelError
	return fmt.Errorf("user not found")
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	if s.logger != nil {
		s.logger = s.logger.With("to", to)
	}
	return nil
}

func (s *Session) Data(r io.Reader) error {
	// parse incoming message
	msg, err := parsemail.Parse(r)
	if err != nil {
		if s.logger != nil {
			s.logger = s.logger.With("dataerror", err)
			s.logLevel = slog.LevelError
		}
		return err
	}

	// grab headers and content
	header := msg.Header
	subject := header.Get("Subject")
	from := s.from
	to := header.Get("To")
	cc := header.Get("Cc")
	bcc := header.Get("Bcc")

	requestBody, err := sendmail.NewMessage(from, to, subject,
		sendmail.WithCc(cc),
		sendmail.WithBcc(bcc),
		sendmail.WithAttachments(msg.Attachments),
		sendmail.WithSaveToSentItems(s.saveToSentItems),
	).SendMailPostRequestBody()
	if err != nil {
		if s.logger != nil {
			s.logger = s.logger.With("dataerror", err)
			s.logLevel = slog.LevelError
		}
		return err
	}
	/* body := models.NewItemBody()
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
			if s.logger != nil {
				s.logger = s.logger.With("dataerror", err)
				s.logLevel = slog.LevelError
			}
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
	requestBody.SetSaveToSentItems(&s.saveToSentItems) */

	// send it
	if err := s.user.SendMail().Post(context.Background(), requestBody, nil); err != nil {
		if s.logger != nil {
			s.logger = s.logger.With("dataerror", err)
			s.logLevel = slog.LevelError
		}
		return err
	}

	if s.logger != nil {
		s.logger = s.logger.With("status", "message sent")
		if s.logLevel < slog.LevelInfo {
			s.logLevel = slog.LevelInfo
		}
	}

	return nil
}

func (s *Session) Reset() {
	if s.logger != nil {
		switch s.logLevel {
		case slog.LevelError:
			s.logger.Error("session ended")
		case slog.LevelInfo:
			s.logger.Info("session ended")
		case slog.LevelWarn:
			s.logger.Warn("session ended")
		}
	}
}

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
