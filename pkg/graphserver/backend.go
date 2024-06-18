package graphserver

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/emersion/go-smtp"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
)

type Backend struct {
	client *graph.GraphServiceClient
}

// NewGraphBackend sets up a new server
func NewGraphBackend(clientId, tenantId, secret string) (*Backend, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantId, clientId, secret, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a cred from a secret: %w", err)
	}

	client, err := graph.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	return &Backend{client}, nil
}

// NewSession is called after client greeting (EHLO, HELO).
func (bkd *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}
