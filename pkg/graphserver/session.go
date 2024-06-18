package graphserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/mail"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

type Session struct {
	user   *users.UserItemRequestBuilder
	client *graph.GraphServiceClient
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

// AuthMechanisms returns a slice of available auth mechanisms; only PLAIN is
// supported in this example.
func (s *Session) AuthMechanisms() []string {
	return []string{}
}

// Auth is the handler for supported authenticators.
func (s *Session) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		if username != "username" || password != "password" {
			return errors.New("invalid username or password")
		}
		return nil
	}), nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	if s.client == nil {
		return fmt.Errorf("graph client not initialised")
	}

	// get UserItemRequestBuilder
	if user := s.client.Users().ByUserId(from); user != nil {
		s.user = user
		return nil
	}

	// see if user exists
	return fmt.Errorf("user not found")
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	log.Println("Rcpt to:", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	// parse incoming message
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}

	// Read the body
	bodyBuffer := new(bytes.Buffer)
	bodyBuffer.ReadFrom(msg.Body)
	bodyString := bodyBuffer.String()
	body := models.NewItemBody()
	body.SetContent(&bodyString)

	header := msg.Header
	subject := header.Get("Subject")

	// build the message
	message := models.NewMessage()
	message.SetBody(body)
	message.SetSubject(&subject)
	message.SetToRecipients(parseAddressList(header.Get("To")))
	message.SetCcRecipients(parseAddressList(header.Get("Cc")))
	message.SetBccRecipients(parseAddressList(header.Get("Bcc")))

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
