package sendmail

import (
	"io"
	"strings"

	"github.com/OfimaticSRL/parsemail"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

type Message struct {
	from            string
	subject         string
	to              string
	cc              string
	bcc             string
	body            string
	attachments     []parsemail.Attachment
	saveToSentItems bool
}

func NewMessage(from, to, subject string, opts ...MessageOption) *Message {
	m := new(Message)

	// apply options
	for _, o := range opts {
		o(m)
	}

	// set mandatory options
	m.from = from
	m.to = to
	m.subject = subject

	return m
}

func (m *Message) SendMailPostRequestBody() (*users.ItemSendmailSendMailPostRequestBody, error) {
	// create email body
	body := models.NewItemBody()
	body.SetContent(&m.body)

	// create message
	message := models.NewMessage()
	message.SetBody(body)
	message.SetSubject(&m.subject)

	if addrs := parseAddressList(m.to); len(addrs) > 0 {
		message.SetToRecipients(addrs)
	}

	if addrs := parseAddressList(m.cc); len(addrs) > 0 {
		message.SetCcRecipients(addrs)
	}

	if addrs := parseAddressList(m.bcc); len(addrs) > 0 {
		message.SetBccRecipients(addrs)
	}

	// add sender/from
	recipient := models.NewRecipient()
	emailAddress := models.NewEmailAddress()
	emailAddress.SetAddress(&m.from)
	recipient.SetEmailAddress(emailAddress)
	message.SetFrom(recipient)

	// handle any attachments
	attachments := []models.Attachmentable{}
	for _, a := range m.attachments {
		data, err := io.ReadAll(a.Data)
		if err != nil {
			return nil, err
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

	// create SendMailPostRequestBody
	requestBody := users.NewItemSendmailSendMailPostRequestBody()
	requestBody.SetMessage(message)
	requestBody.SetSaveToSentItems(&m.saveToSentItems)

	return requestBody, nil
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
