package graphserver

import (
	"fmt"
	"log/slog"
	"net"
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/emersion/go-smtp"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
)

type Backend struct {
	client          *graph.GraphServiceClient
	debug           bool
	saveToSentItems bool
	logger          *slog.Logger
	allowedSenders  []string
	allowedSources  []string
}

// NewGraphBackend sets up a new server
func NewGraphBackend(clientId, tenantId, secret string, opts ...BackendOption) (*Backend, error) {
	return newbackend(clientId, tenantId, secret, opts...)
}

// NewDebugGraphBackend sets up a new server
func NewDebugGraphBackend(clientId, tenantId, secret string, opts ...BackendOption) (*Backend, error) {
	return newbackend(clientId, tenantId, secret, opts...)
}

func newbackend(clientId, tenantId, secret string, opts ...BackendOption) (*Backend, error) {
	b := new(Backend)

	// apply options
	for _, o := range opts {
		o(b)
	}

	// create graph client
	cred, err := azidentity.NewClientSecretCredential(tenantId, clientId, secret, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a cred from a secret: %w", err)
	}

	client, err := graph.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
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
	logger := b.logger
	if logger != nil {
		logger = logger.With("helo", c.Hostname()).With("remote", c.Conn().RemoteAddr().String())
	}

	// Check if IP is allowed
	if len(b.allowedSources) > 0 {
		if addr, _, err := net.SplitHostPort(c.Conn().RemoteAddr().String()); err == nil {
			if _, found := slices.BinarySearch(b.allowedSources, addr); !found {
				return nil, fmt.Errorf("source not allowed")
			}
		}
	}

	return &Session{
		client:          b.client,
		debug:           b.debug,
		logger:          logger,
		allowedSenders:  b.allowedSenders,
		saveToSentItems: b.saveToSentItems,
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

func WithLogger(logger *slog.Logger) BackendOption {
	return func(b *Backend) {
		b.logger = logger
	}
}
