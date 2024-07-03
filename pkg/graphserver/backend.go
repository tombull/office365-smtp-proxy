package graphserver

import (
	"fmt"
	"net"
	"slices"

	"github.com/andrewheberle/graph-smtpd/pkg/graphclient"
	"github.com/emersion/go-smtp"
)

type Backend struct {
	client          *graphclient.Client
	saveToSentItems bool
	logger          Logger
	allowedSenders  []string
	allowedSources  []string
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

	return b, nil
}

// NewSession is called after client greeting (EHLO, HELO).
func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	// Check if IP is allowed
	if len(b.allowedSources) > 0 {
		if addr, _, err := net.SplitHostPort(c.Conn().RemoteAddr().String()); err == nil {
			if _, found := slices.BinarySearch(b.allowedSources, addr); !found {
				return nil, fmt.Errorf("source not allowed")
			}
		}
	}

	// return new session
	return &Session{
		client:          b.client,
		logger:          b.logger,
		allowedSenders:  b.allowedSenders,
		saveToSentItems: b.saveToSentItems,
		helo:            c.Hostname(),
		remote:          c.Conn().RemoteAddr().String(),
		errors:          make([]error, 0),
	}, nil
}

type BackendOption func(*Backend)

func WithSaveToSentItems(save bool) BackendOption {
	return func(b *Backend) {
		b.saveToSentItems = save
	}
}

func WithAllowedSenders(senders []string) BackendOption {
	return func(b *Backend) {
		if senders == nil {
			b.allowedSenders = make([]string, 0)
		} else {
			// make sure it's sorted
			slices.Sort(senders)
			b.allowedSenders = senders
		}
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
