package graphserver

import (
	"fmt"
	"net"
	"net/mail"
	"slices"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tombull/office365-smtp-proxy/pkg/graphclient"
)

type Backend struct {
	client         *graphclient.Client
	logger         Logger
	allowedSenders []string
	allowedSources []string
	sendUser       string

	reg prometheus.Registerer

	// metrics
	emailTotal prometheus.Counter
	sendErrors prometheus.Counter
	sendDenied prometheus.Counter
}

// NewGraphBackend sets up a new server
func NewGraphBackend(clientId, tenantId, secret string, opts ...BackendOption) (*Backend, error) {
	return newbackend(clientId, tenantId, secret, opts...)
}

func newbackend(clientId, tenantId, secret string, opts ...BackendOption) (*Backend, error) {
	b := new(Backend)

	// apply options
	for _, o := range opts {
		o(b)
	}

	// create graph client
	client, err := graphclient.NewClient(tenantId, clientId, secret)
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	b.client = client

	// set defaults
	if b.allowedSenders == nil {
		b.allowedSenders = make([]string, 0)
	}

	if b.allowedSources == nil {
		b.allowedSources = make([]string, 0)
	}

	normalizedSenders, err := normalizeMailboxList(b.allowedSenders)
	if err != nil {
		return nil, err
	}
	b.allowedSenders = normalizedSenders

	if b.sendUser != "" {
		normalized, err := normalizeMailbox(b.sendUser)
		if err != nil {
			return nil, fmt.Errorf("invalid senduser %q: %w", b.sendUser, err)
		}
		b.sendUser = normalized
	}

	// set up metrics
	b.emailTotal = promauto.With(b.reg).NewCounter(
		prometheus.CounterOpts{
			Name: "office365_smtp_proxy_email_total",
			Help: "Total number of emails accepted by SMTP",
		},
	)
	b.sendErrors = promauto.With(b.reg).NewCounter(
		prometheus.CounterOpts{
			Name: "office365_smtp_proxy_email_errors_total",
			Help: "Total number of send errors",
		},
	)
	b.sendDenied = promauto.With(b.reg).NewCounter(
		prometheus.CounterOpts{
			Name: "office365_smtp_proxy_email_denied_total",
			Help: "Total number of emails denied",
		},
	)

	return b, nil
}

// NewSession is called after client greeting (EHLO, HELO).
func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	// Check if IP is allowed
	if len(b.allowedSources) > 0 {
		if addr, _, err := net.SplitHostPort(c.Conn().RemoteAddr().String()); err == nil {
			if _, found := slices.BinarySearch(b.allowedSources, addr); !found {
				b.sendDenied.Inc()
				return nil, fmt.Errorf("source not allowed")
			}
		}
	}

	// increment total metric
	b.emailTotal.Inc()

	// return new session
	return &Session{
		client:         b.client,
		logger:         b.logger,
		allowedSenders: b.allowedSenders,
		sendUser:       b.sendUser,
		helo:           c.Hostname(),
		remote:         c.Conn().RemoteAddr().String(),
		errors:         make([]error, 0),
		sendErrors:     b.sendErrors,
		sendDenied:     b.sendDenied,
	}, nil
}

type BackendOption func(*Backend)

func WithAllowedSenders(senders []string) BackendOption {
	return func(b *Backend) {
		if senders == nil {
			b.allowedSenders = make([]string, 0)
		} else {
			b.allowedSenders = append([]string(nil), senders...)
		}
	}
}

func WithSendUser(sendUser string) BackendOption {
	return func(b *Backend) {
		b.sendUser = strings.TrimSpace(sendUser)
	}
}

func WithAllowedSources(sources []string) BackendOption {
	return func(b *Backend) {
		if sources == nil {
			b.allowedSources = make([]string, 0)
		} else {
			// make sure it's sorted
			slices.Sort(sources)
			b.allowedSources = sources
		}
	}
}

func WithLogger(logger Logger) BackendOption {
	return func(b *Backend) {
		b.logger = logger
	}
}

func WithPrometheusRegistry(reg *prometheus.Registry) BackendOption {
	return func(b *Backend) {
		b.reg = reg
	}
}

func normalizeMailbox(address string) (string, error) {
	parsed, err := mail.ParseAddress(strings.TrimSpace(address))
	if err != nil {
		return "", err
	}

	return strings.ToLower(parsed.Address), nil
}

func normalizeMailboxList(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		addresses, err := mail.ParseAddressList(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed sender %q: %w", value, err)
		}

		for _, address := range addresses {
			normalized = append(normalized, strings.ToLower(address.Address))
		}
	}

	slices.Sort(normalized)
	return normalized, nil
}
