package graphserver

import (
	"fmt"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/emersion/go-smtp"
)

type Backend struct {
	client confidential.Client
}

// NewGraphBackend sets up a new server
func NewGraphBackend(clientId, tenantId, secret string) (*Backend, error) {
	cred, err := confidential.NewCredFromSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("could not create a cred from a secret: %w", err)
	}

	client, err := confidential.New(fmt.Sprintf("https://login.microsoftonline.com/%s", tenantId), clientId, cred)
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	return &Backend{client}, nil
}

// NewSession is called after client greeting (EHLO, HELO).
func (bkd *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}
