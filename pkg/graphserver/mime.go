package graphserver

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"sort"
	"strings"
)

const maxGraphMIMEEncodedBytes = int(3.75 * 1024 * 1024)

func prepareGraphMIME(raw []byte, from string, recipients []string) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("message data was empty")
	}

	msg, body, err := parseAndValidateMIME(raw)
	if err != nil {
		return nil, err
	}

	headers := cloneHeader(msg.Header)
	setHeader(headers, "From", from)
	setHeader(headers, "To", strings.Join(recipients, ", "))
	delete(headers, "Cc")
	delete(headers, "Bcc")
	delete(headers, "Sender")
	delete(headers, "Return-Path")

	payload, err := marshalMIME(headers, body)
	if err != nil {
		return nil, err
	}

	if _, _, err := parseAndValidateMIME(payload); err != nil {
		return nil, fmt.Errorf("final MIME payload was invalid: %w", err)
	}

	if encodedLen := base64.StdEncoding.EncodedLen(len(payload)); encodedLen > maxGraphMIMEEncodedBytes {
		return nil, fmt.Errorf("base64 encoded MIME payload size %d exceeds limit %d", encodedLen, maxGraphMIMEEncodedBytes)
	}

	return payload, nil
}

func parseAndValidateMIME(raw []byte) (*mail.Message, []byte, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid MIME headers: %w", err)
	}

	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read MIME body: %w", err)
	}

	if err := validateMIMEEntity(msg.Header, body); err != nil {
		return nil, nil, err
	}

	return msg, body, nil
}

func validateMIMEEntity(header mail.Header, body []byte) error {
	contentType := header.Get("Content-Type")
	if contentType == "" {
		return nil
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("invalid Content-Type %q: %w", contentType, err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return fmt.Errorf("multipart message missing boundary")
		}

		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("invalid multipart body: %w", err)
			}

			partBody, err := io.ReadAll(part)
			if err != nil {
				return fmt.Errorf("could not read multipart section %q: %w", part.FileName(), err)
			}

			partHeader := mail.Header(part.Header)
			if nestedContentType := partHeader.Get("Content-Type"); nestedContentType != "" {
				parsedType, _, err := mime.ParseMediaType(nestedContentType)
				if err != nil {
					return fmt.Errorf("invalid nested Content-Type %q: %w", nestedContentType, err)
				}

				if parsedType == "message/rfc822" {
					if _, _, err := parseAndValidateMIME(partBody); err != nil {
						return err
					}
					continue
				}
			}

			if err := validateMIMEEntity(partHeader, partBody); err != nil {
				return err
			}
		}
	}

	return nil
}

func cloneHeader(header mail.Header) mail.Header {
	clone := make(mail.Header, len(header))
	for key, values := range header {
		clone[key] = append([]string(nil), values...)
	}

	return clone
}

func setHeader(header mail.Header, key, value string) {
	header[key] = []string{value}
}

func marshalMIME(header mail.Header, body []byte) ([]byte, error) {
	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, key := range keys {
		for _, value := range header[key] {
			if strings.ContainsAny(value, "\r\n") {
				return nil, fmt.Errorf("header %q contained invalid newline characters", key)
			}
			if _, err := fmt.Fprintf(&buf, "%s: %s\r\n", key, value); err != nil {
				return nil, fmt.Errorf("could not write MIME header %q: %w", key, err)
			}
		}
	}

	if _, err := buf.WriteString("\r\n"); err != nil {
		return nil, fmt.Errorf("could not terminate MIME headers: %w", err)
	}

	if _, err := buf.Write(body); err != nil {
		return nil, fmt.Errorf("could not write MIME body: %w", err)
	}

	return buf.Bytes(), nil
}
