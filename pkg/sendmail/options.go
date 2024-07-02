package sendmail

import "github.com/OfimaticSRL/parsemail"

type MessageOption func(*Message)

func WithCc(cc string) MessageOption {
	return func(m *Message) {
		m.cc = cc
	}
}

func WithBcc(bcc string) MessageOption {
	return func(m *Message) {
		m.bcc = bcc
	}
}

func WithBody(body string) MessageOption {
	return func(m *Message) {
		m.body = body
	}
}

func WithAttachments(attachments []parsemail.Attachment) MessageOption {
	return func(m *Message) {
		m.attachments = attachments
	}
}

func WithSaveToSentItems(save bool) MessageOption {
	return func(m *Message) {
		m.saveToSentItems = save
	}
}
