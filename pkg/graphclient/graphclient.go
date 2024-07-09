package graphclient

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
)

type Client struct {
	graph.GraphServiceClient
}

// NewClient creates a new Graph API client
func NewClient(tenantid, clientid, secret string) (*Client, error) {
	// error checking
	if tenantid == "" || clientid == "" || secret == "" {
		return nil, fmt.Errorf("tenantid, clientid and secret must not be blank")
	}

	// create graph client
	cred, err := azidentity.NewClientSecretCredential(tenantid, clientid, secret, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create cred from secret: %w", err)
	}

	client, err := graph.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	return &Client{*client}, nil
}
