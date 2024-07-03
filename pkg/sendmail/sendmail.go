package sendmail

import (
	"strings"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

type Message struct {
	body        *models.ItemBody
	message     *models.Message
	requestBody *users.ItemSendmailSendMailPostRequestBody
}

func NewMessage(from, to, subject string, opts ...MessageOption) *Message {
	m := new(Message)

	m.body = models.NewItemBody()
	m.message = models.NewMessage()
	m.requestBody = users.NewItemSendmailSendMailPostRequestBody()

	// apply options
	for _, o := range opts {
		o(m)
	}

	// set subject and message body
	m.message.SetSubject(&subject)
	m.message.SetBody(m.body)

	// add sender/from
	recipient := models.NewRecipient()
	emailAddress := models.NewEmailAddress()
	emailAddress.SetAddress(&from)
	recipient.SetEmailAddress(emailAddress)
	m.message.SetFrom(recipient)

	// set recipients
	if addrs := parseAddressList(to); len(addrs) > 0 {
		m.message.SetToRecipients(addrs)
	}

	return m
}

func (m *Message) SendMailPostRequestBody() (*users.ItemSendmailSendMailPostRequestBody, error) {
	// create SendMailPostRequestBody
	m.requestBody.SetMessage(m.message)

	return m.requestBody, nil
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
