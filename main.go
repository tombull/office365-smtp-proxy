package main

import (
	"log"

	"github.com/andrewheberle/graph-smtpd/pkg/graphserver"
	"github.com/emersion/go-smtp"
)

func main() {
	be := &graphserver.Backend{}

	s := smtp.NewServer(be)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
