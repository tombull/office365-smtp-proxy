package graphserver

import (
	"encoding/base64"
	"net/mail"
	"strings"
	"testing"
)

func TestPrepareGraphMIMEOverridesEnvelopeHeaders(t *testing.T) {
	raw := strings.Join([]string{
		"From: Original Sender <original@example.com>",
		"To: Old Recipient <old@example.com>",
		"Cc: copied@example.com",
		"Bcc: hidden@example.com",
		"Subject: Multipart test",
		"MIME-Version: 1.0",
		"Content-Type: multipart/mixed; boundary=outer",
		"",
		"--outer",
		"Content-Type: multipart/alternative; boundary=inner",
		"",
		"--inner",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"plain text",
		"--inner",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<p>html body</p>",
		"--inner--",
		"--outer",
		"Content-Type: text/csv",
		"Content-Disposition: attachment; filename=report.csv",
		"",
		"name,status",
		"multipart,ok",
		"--outer--",
		"",
	}, "\r\n")

	payload, err := prepareGraphMIME([]byte(raw), "test1@example.com", []string{"test1@example.com", "test2@example.com"})
	if err != nil {
		t.Fatalf("prepareGraphMIME() error = %v", err)
	}

	msg, err := mail.ReadMessage(strings.NewReader(string(payload)))
	if err != nil {
		t.Fatalf("mail.ReadMessage() error = %v", err)
	}

	if got := msg.Header.Get("From"); got != "test1@example.com" {
		t.Fatalf("From header = %q, want %q", got, "test1@example.com")
	}

	if got := msg.Header.Get("To"); got != "test1@example.com, test2@example.com" {
		t.Fatalf("To header = %q, want %q", got, "test1@example.com, test2@example.com")
	}

	if got := msg.Header.Get("Cc"); got != "" {
		t.Fatalf("Cc header = %q, want empty", got)
	}

	if got := msg.Header.Get("Bcc"); got != "" {
		t.Fatalf("Bcc header = %q, want empty", got)
	}

	if !strings.Contains(string(payload), "boundary=outer") || !strings.Contains(string(payload), "boundary=inner") {
		t.Fatalf("payload did not preserve multipart boundaries")
	}
}

func TestPrepareGraphMIMERejectsInvalidMultipart(t *testing.T) {
	raw := strings.Join([]string{
		"From: original@example.com",
		"To: old@example.com",
		"Subject: Broken multipart",
		"MIME-Version: 1.0",
		"Content-Type: multipart/mixed; boundary=outer",
		"",
		"--outer",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"unterminated multipart body",
	}, "\r\n")

	if _, err := prepareGraphMIME([]byte(raw), "test1@example.com", []string{"test1@example.com"}); err == nil {
		t.Fatal("prepareGraphMIME() error = nil, want malformed multipart error")
	}
}

func TestPrepareGraphMIMERejectsOversizedEncodedPayload(t *testing.T) {
	bodyLen := maxGraphMIMEEncodedBytes
	bodyLen = bodyLen - (bodyLen % 4) + 8
	oversizedBody := strings.Repeat("a", bodyLen)
	raw := strings.Join([]string{
		"From: original@example.com",
		"To: old@example.com",
		"Subject: Large message",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		oversizedBody,
	}, "\r\n")

	_, err := prepareGraphMIME([]byte(raw), "test1@example.com", []string{"test1@example.com"})
	if err == nil {
		t.Fatal("prepareGraphMIME() error = nil, want size rejection")
	}

	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("prepareGraphMIME() error = %v, want size limit failure", err)
	}

	encodedLen := base64.StdEncoding.EncodedLen(len([]byte(raw)))
	if encodedLen <= maxGraphMIMEEncodedBytes {
		t.Fatalf("test setup invalid: encoded length %d did not exceed limit %d", encodedLen, maxGraphMIMEEncodedBytes)
	}
}

func TestNormalizeMailboxListParsesCommaSeparatedEnvValue(t *testing.T) {
	got, err := normalizeMailboxList([]string{"invoices@porkpiewifi.com,billing@porkpiewifi.com,website@porkpiewifi.com"})
	if err != nil {
		t.Fatalf("normalizeMailboxList() error = %v", err)
	}

	want := []string{
		"billing@porkpiewifi.com",
		"invoices@porkpiewifi.com",
		"website@porkpiewifi.com",
	}

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("normalizeMailboxList() = %v, want %v", got, want)
	}
}
