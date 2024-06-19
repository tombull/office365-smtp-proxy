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
	client         *graph.GraphServiceClient
	debug          bool
	SessionLog     *slog.Logger
	allowedSenders []string
	allowedSources []string
}

// NewGraphBackend sets up a new server
func NewGraphBackend(clientId, tenantId, secret string) (*Backend, error) {
	return newbackend(clientId, tenantId, secret, false)
}

// NewDebugGraphBackend sets up a new server
func NewDebugGraphBackend(clientId, tenantId, secret string) (*Backend, error) {
	return newbackend(clientId, tenantId, secret, true)
}

func newbackend(clientId, tenantId, secret string, debug bool) (*Backend, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantId, clientId, secret, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a cred from a secret: %w", err)
	}

	client, err := graph.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	return &Backend{client, debug, nil, make([]string, 0), make([]string, 0)}, nil
}

// NewSession is called after client greeting (EHLO, HELO).
func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	logger := b.SessionLog
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
		client:         b.client,
		debug:          b.debug,
		SessionLog:     logger,
		allowedSenders: b.allowedSenders,
	}, nil
}

func (b *Backend) SetAllowedSenders(senders []string) {
	if senders == nil {
		b.allowedSenders = make([]string, 0)
	} else {
		// make sure it's sorted
		slices.Sort(senders)
		b.allowedSenders = senders
	}
}

func (b *Backend) SetAllowedSources(sources []string) {
	if sources == nil {
		b.allowedSources = make([]string, 0)
	} else {
		// make sure it's sorted
		slices.Sort(sources)
		b.allowedSources = sources
	}
}
