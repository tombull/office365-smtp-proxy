package graphclient

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	odataerrors "github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
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

// SendMime sends a fully formed RFC822 MIME message through Microsoft Graph by
// creating a MIME draft, patching the From address, and then sending the draft.
func (c *Client) SendMime(ctx context.Context, graphUserID, fromAddress string, mimeMessage []byte) error {
	graphUserID = strings.TrimSpace(graphUserID)
	if graphUserID == "" {
		return fmt.Errorf("graphUserID must not be blank")
	}

	fromAddress = strings.TrimSpace(fromAddress)
	if fromAddress == "" {
		return fmt.Errorf("fromAddress must not be blank")
	}

	if len(mimeMessage) == 0 {
		return fmt.Errorf("mime message must not be empty")
	}

	draft, err := c.createMimeDraft(ctx, graphUserID, mimeMessage)
	if err != nil {
		return fmt.Errorf("could not create MIME draft: %w", err)
	}

	draftID := draft.GetId()
	if draftID == nil || strings.TrimSpace(*draftID) == "" {
		return fmt.Errorf("graph did not return a draft message id")
	}

	if err := c.patchDraftFrom(ctx, graphUserID, *draftID, fromAddress); err != nil {
		return fmt.Errorf("could not patch draft from address: %w", err)
	}

	if err := c.sendDraft(ctx, graphUserID, *draftID); err != nil {
		return fmt.Errorf("could not send draft message: %w", err)
	}

	return nil
}

func (c *Client) createMimeDraft(ctx context.Context, userID string, mimeMessage []byte) (graphmodels.Messageable, error) {
	builder := c.Users().ByUserId(userID).Messages()
	requestInfo := abstractions.NewRequestInformationWithMethodAndUrlTemplateAndPathParameters(
		abstractions.POST,
		builder.UrlTemplate,
		builder.PathParameters,
	)
	requestInfo.Headers.TryAdd("Accept", "application/json")
	requestInfo.SetStreamContentAndContentType(base64Encoded(mimeMessage), "text/plain")

	errorMapping := abstractions.ErrorMappings{
		"4XX": odataerrors.CreateODataErrorFromDiscriminatorValue,
		"5XX": odataerrors.CreateODataErrorFromDiscriminatorValue,
	}

	res, err := c.GetAdapter().Send(ctx, requestInfo, graphmodels.CreateMessageFromDiscriminatorValue, errorMapping)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("graph returned no draft message")
	}

	return res.(graphmodels.Messageable), nil
}

func (c *Client) patchDraftFrom(ctx context.Context, userID, messageID, fromAddress string) error {
	message := graphmodels.NewMessage()
	fromRecipient := graphmodels.NewRecipient()
	fromEmail := graphmodels.NewEmailAddress()
	fromEmail.SetAddress(&fromAddress)
	fromRecipient.SetEmailAddress(fromEmail)
	message.SetFrom(fromRecipient)

	builder := c.Users().ByUserId(userID).Messages().ByMessageId(messageID)
	_, err := builder.Patch(ctx, message, nil)
	return err
}

func (c *Client) sendDraft(ctx context.Context, userID, messageID string) error {
	return c.Users().ByUserId(userID).Messages().ByMessageId(messageID).Send().Post(ctx, nil)
}

func base64Encoded(content []byte) []byte {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(content)))
	base64.StdEncoding.Encode(encoded, content)
	return encoded
}
