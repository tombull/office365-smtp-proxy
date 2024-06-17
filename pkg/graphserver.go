package graphserver

import "github.com/emersion/go-smtp"

type Backend struct{}

type Session struct{}

// NewSession is called after client greeting (EHLO, HELO).
func (bkd *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}
