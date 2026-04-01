package graphserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tombull/office365-smtp-proxy/pkg/graphclient"
)

type Session struct {
	from           string
	recipients     []string
	graphUser      string
	client         *graphclient.Client
	logger         Logger
	logLevel       Level
	allowedSenders []string
	sendUser       string
	helo           string
	remote         string
	errors         []error
	status         string

	sendErrors prometheus.Counter
	sendDenied prometheus.Counter
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	if s.client == nil {
		return s.fail(errors.New("graph client not initialised"), false)
	}

	normalizedFrom, err := normalizeMailbox(from)
	if err != nil {
		return s.fail(fmt.Errorf("invalid MAIL FROM address %q: %w", from, err), true)
	}
	s.from = normalizedFrom
	s.graphUser = s.from
	if s.sendUser != "" {
		s.graphUser = s.sendUser
	}

	// check that sender is allowed
	if len(s.allowedSenders) > 0 {
		if _, found := slices.BinarySearch(s.allowedSenders, s.from); !found {
			return s.fail(fmt.Errorf("sender %q not allowed", s.from), true)
		}
	}

	s.recipients = s.recipients[:0]
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	normalizedTo, err := normalizeMailbox(to)
	if err != nil {
		return s.fail(fmt.Errorf("invalid RCPT TO address %q: %w", to, err), true)
	}

	s.recipients = append(s.recipients, normalizedTo)

	return nil
}

func (s *Session) Data(r io.Reader) error {
	if s.from == "" {
		return s.fail(errors.New("message missing MAIL FROM envelope"), true)
	}

	if len(s.recipients) == 0 {
		return s.fail(errors.New("message missing RCPT TO recipients"), true)
	}

	rawMessage, err := io.ReadAll(r)
	if err != nil {
		return s.fail(fmt.Errorf("could not read message data: %w", err), false)
	}

	payload, err := prepareGraphMIME(rawMessage, s.from, s.recipients)
	if err != nil {
		return s.fail(fmt.Errorf("rejected MIME message: %w", err), true)
	}

	if err := s.client.SendMime(context.Background(), s.graphUser, s.from, payload); err != nil {
		return s.fail(fmt.Errorf("error sending MIME message: %w", err), false)
	}

	s.status = "message sent"
	if s.logLevel < LevelInfo {
		s.logLevel = LevelInfo
	}

	return nil
}

func (s *Session) Reset() {
	if s.logger != nil {
		to := strings.Join(s.recipients, ",")
		switch s.logLevel {
		case LevelError:
			s.logger.Error("session ended", "errors", s.errors, "from", s.from, "graph_user", s.graphUser, "to", to)
		case LevelInfo:
			s.logger.Info("session ended", "status", s.status, "from", s.from, "graph_user", s.graphUser, "to", to)
		case LevelWarn:
			s.logger.Warn("session ended", "status", s.status, "from", s.from, "graph_user", s.graphUser, "to", to)
		}
	}

	s.from = ""
	s.recipients = s.recipients[:0]
	s.graphUser = ""
	s.errors = s.errors[:0]
	s.status = ""
	s.logLevel = LevelInfo
}

func (s *Session) Logout() error {
	return nil
}

func (s *Session) fail(err error, denied bool) error {
	s.errors = append(s.errors, err)
	s.logLevel = LevelError
	if denied {
		s.sendDenied.Inc()
	} else {
		s.sendErrors.Inc()
	}

	return err
}
