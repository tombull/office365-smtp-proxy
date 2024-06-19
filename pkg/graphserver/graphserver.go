package graphserver

import (
	"log/slog"

	graph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

type Backend struct {
	client         *graph.GraphServiceClient
	debug          bool
	SessionLog     *slog.Logger
	allowedSenders []string
	allowedSources []string
}

type Session struct {
	from           string
	user           *users.UserItemRequestBuilder
	client         *graph.GraphServiceClient
	debug          bool
	SessionLog     *slog.Logger
	logLevel       slog.Level
	allowedSenders []string
}
