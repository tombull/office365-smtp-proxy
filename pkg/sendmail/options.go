package sendmail

import (
	"io"

	"github.com/OfimaticSRL/parsemail"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

type MessageOption func(*Message)

func WithCc(cc string) MessageOption {
	return func(m *Message) {
		if addrs := parseAddressList(cc); len(addrs) > 0 {
			m.message.SetCcRecipients(addrs)
		}
	}
}

func WithBcc(bcc string) MessageOption {
	return func(m *Message) {
		if addrs := parseAddressList(bcc); len(addrs) > 0 {
			m.message.SetBccRecipients(addrs)
		}
	}
}

func WithBody(body string) MessageOption {
	return func(m *Message) {
		m.body.SetContent(&body)
	}
}

func WithAttachments(attachments []parsemail.Attachment) MessageOption {
	return func(m *Message) {
		// handle any attachments
		attachmentable := []models.Attachmentable{}
		for _, a := range attachments {
			data, err := io.ReadAll(a.Data)
			if err != nil {
				// we cannot recover from this
				panic(err)
			}
			attachment := models.NewFileAttachment()
			attachment.SetName(&a.Filename)
			attachment.SetContentType(&a.ContentType)
			attachment.SetContentBytes(data)

			// add to attachmentsList
			attachmentable = append(attachmentable, attachment)
		}

		// add if any attachments
		if len(attachments) > 0 {
			m.message.SetAttachments(attachmentable)
		}
	}
}

func WithSaveToSentItems(save bool) MessageOption {
	return func(m *Message) {
		m.saveToSentItems = save
	}
}

func WithHTMLContent() MessageOption {
	return func(m *Message) {
		contentType := models.HTML_BODYTYPE
		m.body.SetContentType(&contentType)
	}
}
