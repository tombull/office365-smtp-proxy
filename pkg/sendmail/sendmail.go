package sendmail

import (
	"context"
	"strings"

	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	graphusers "github.com/microsoftgraph/msgraph-sdk-go/users"
)

type Message struct {
	body        *graphmodels.ItemBody
	message     *graphmodels.Message
	requestBody *graphusers.ItemSendMailPostRequestBody
}

func NewMessage(from, to, subject string, opts ...MessageOption) *Message {
	m := new(Message)

	m.body = graphmodels.NewItemBody()
	m.message = graphmodels.NewMessage()
	m.requestBody = graphusers.NewItemSendMailPostRequestBody()

	// apply options
	for _, o := range opts {
		o(m)
	}

	// set subject and message body
	m.message.SetSubject(&subject)
	m.message.SetBody(m.body)

	// add sender/from
	recipient := graphmodels.NewRecipient()
	emailAddress := graphmodels.NewEmailAddress()
	emailAddress.SetAddress(&from)
	recipient.SetEmailAddress(emailAddress)
	m.message.SetFrom(recipient)

	// set recipients
	if addrs := parseAddressList(to); len(addrs) > 0 {
		m.message.SetToRecipients(addrs)
	}

	return m
}

func (m *Message) Send(ctx context.Context, user *graphusers.UserItemRequestBuilder) error {
	// create SendMailPostRequestBody
	m.requestBody.SetMessage(m.message)

	return user.SendMail().Post(ctx, m.requestBody, nil)
}

func parseAddressList(addresses string) []graphmodels.Recipientable {
	recipientList := []graphmodels.Recipientable{}

	if addresses == "" {
		return recipientList
	}

	// Split the address list by commas and trim spaces
	list := strings.Split(addresses, ",")
	for i := range list {
		address := strings.TrimSpace(list[i])

		// build recipient
		recipient := graphmodels.NewRecipient()
		emailAddress := graphmodels.NewEmailAddress()
		emailAddress.SetAddress(&address)
		recipient.SetEmailAddress(emailAddress)

		// add to list
		recipientList = append(recipientList, recipient)
	}

	return recipientList
}
