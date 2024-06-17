package graphserver

import "github.com/emersion/go-smtp"

type Backend struct {
	clientId string
	tenantId string
	secret   string
}

// NewGraphBackend sets up a new server
func NewGraphBackend(clientId, tenantId, secret string) *Backend {
	return &Backend{clientId, tenantId, secret}
}

// NewSession is called after client greeting (EHLO, HELO).
func (bkd *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}
