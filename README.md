# office365-smtp-proxy

This service is a SMTP daemon that will accept emails via SMTP and then submit them via the Microsoft Graph API to Microsoft 365, however in theory the submission process could be generalised to support any API driven email submission service.

The idea is to allow this service to run locally and accept emails from devices/systems that cannot support the modern authentication requirements of Exchange Online, such as OAuth2, and then relay them securely to Microsoft 365.

## Azure AD/Entra ID Requirements

1. Create an "App Registration"
2. Note down the Client/Application ID and Tenant ID
3. Create a Client Secret and record the value
4. Set permissions (see below)

### Permissions

The App Registration requires both the `Mail.ReadWrite` and `Mail.Send` Microsoft Graph application permissions, and those permissions must have admin consent granted in your tenant.

This will allow the service to send as any user in your environment.

To limit this ability to specific mailboxes/senders, it is possible to implement an `ApplicationAccessPolicy` to control this as follows:

1. Create a new mail enabled security group (or use an existing one)
2. Add the mailboxes that the service is allowed to send as to the mail enabled security group
3. Create the policy using Exchange Online PowerShell:

    ```powershell
    New-ApplicationAccessPolicy -AppId <APPLICATION_ID> -PolicyScopeGroupId <GROUP_EMAIL_ADDRESS> -AccessRight RestrictAccess -Description <DESCRIPTION>
    ```

This will limit the service to being able to send only as the members of the provided group.

## Running

### Docker

```sh
docker run -p 25:2525 \
    -e OFFICE365_SMTP_PROXY_CLIENTID="clientid" \
    -e OFFICE365_SMTP_PROXY_SECRET="secret" \
    -e OFFICE365_SMTP_PROXY_TENANTID="tenantid" \
    ghcr.io/tombull/office365-smtp-proxy:latest
```

This container accepts SMTP messages, validates the MIME payload, preserves the MIME structure, creates a draft message in Microsoft Graph using MIME, patches the draft `from` address, and then sends the draft. This flow always saves the message to Sent Items, so there is no SMTP server option to disable that behaviour.

### Command-Line Options

* `--addr`: Listen address (default = "localhost:2525") (string)
* `--cert`: Certificate for enabling STARTTLS (string)
* `--clientid`: Client/Application ID (string)
* `--key`: Private key for enabling STARTTLS (string)
* `--secret`: Client Secret (string)
* `--senders`: Allowed senders ([]string)
* `--senduser`: Force Microsoft Graph to send every message as this user ID/email address (string)
* `--sources`: Allowed source IP addresses ([]string)
* `--tenantid`: Tenant ID (string)
* `--metrics`: Listen address for metrics (string)

All command line options may be specified as environment variables in the form of `OFFICE365_SMTP_PROXY_<option>`, with the additional option to supply `OFFICE365_SMTP_PROXY_SECRET_FILE` to allow loading of the client secret from a file.

### MIME-Preserving Behaviour

The SMTP server now forwards the message to Microsoft Graph as preserved MIME rather than rebuilding the message as Graph JSON. That means multipart bodies, HTML alternatives, attachments, and other MIME sections are kept intact as long as the MIME payload is valid.

The Graph submission flow is:

1. Create a draft message using MIME.
2. Patch the draft message `from` address explicitly.
3. Send the draft message.

Before the message is sent to Graph, the SMTP server:

1. Validates that the incoming MIME message is correctly formatted.
2. Rejects malformed or partially readable MIME payloads and logs the failure.
3. Rewrites the MIME envelope-facing headers from the SMTP transaction.
4. Rejects messages whose final MIME payload would exceed the single-request Graph MIME limit once Base64 encoded.

The effective size guard is 3.75 MiB after Base64 encoding. Messages above that limit are rejected before they are submitted to Graph.

### Envelope Handling

The SMTP envelope is authoritative.

* `MAIL FROM` replaces the MIME `From` header.
* The SMTP recipients replace the MIME `To` header.
* Existing MIME `Cc`, `Bcc`, `Sender`, and `Return-Path` headers are removed before submission.
* Invalid envelope addresses cause the SMTP transaction to be rejected and logged.

### Allowed Senders

The `senders` option restricts which SMTP `MAIL FROM` addresses the server will accept. If `OFFICE365_SMTP_PROXY_SENDERS` is set, any message from an envelope sender not in that list is rejected before submission to Graph.

For environment variables and `.env` files, set `OFFICE365_SMTP_PROXY_SENDERS` as a single comma-separated list of mailbox addresses.

Example with multiple allowed senders:

```sh
docker run -p 25:2525 \
    -e OFFICE365_SMTP_PROXY_CLIENTID="clientid" \
    -e OFFICE365_SMTP_PROXY_SECRET="secret" \
    -e OFFICE365_SMTP_PROXY_TENANTID="tenantid" \
    -e OFFICE365_SMTP_PROXY_SENDERS="printer1@example.com,scanner1@example.com,alerts@example.com" \
    ghcr.io/tombull/office365-smtp-proxy:latest
```

The values should be mailbox addresses. They are matched against the SMTP envelope sender, not the original MIME header. Repeated command-line flags and config-file arrays also work, but `.env` usage should be a single comma-separated string.

### Forced Graph Send User

The `senduser` option forces the Graph API call to use a single mailbox for every relayed message, regardless of the SMTP `MAIL FROM` address. This can be set with `--senduser` or `OFFICE365_SMTP_PROXY_SENDUSER`.

Example:

```sh
docker run -p 25:2525 \
    -e OFFICE365_SMTP_PROXY_CLIENTID="clientid" \
    -e OFFICE365_SMTP_PROXY_SECRET="secret" \
    -e OFFICE365_SMTP_PROXY_TENANTID="tenantid" \
    -e OFFICE365_SMTP_PROXY_SENDUSER="relay-user@example.com" \
    ghcr.io/tombull/office365-smtp-proxy:latest
```

Important details:

* `senduser` changes the mailbox used in the Graph API call.
* `senduser` does not rewrite the MIME `From` header beyond the normal SMTP envelope override. The visible `From` still comes from the SMTP transaction.
* If `senduser` is set, the `MAIL FROM` value supplied to the SMTP server must either be the same mailbox as `senduser` or a mailbox that `senduser` is allowed to send as or send on behalf of.
* If your tenant uses an `ApplicationAccessPolicy`, the forced send user must also be within the allowed scope for the application.

### Configuration File

All configuration options may be provided in a YAML or JSON configuration file using the `--config` command-line option or if this is not set, will be looked for in the current working directory as `config.yaml`.

## CLI "sendmail" mode

This is a "sendmail-ish" command line tool.

### Building

```sh
go install github.com/tombull/office365-smtp-proxy/cmd/graph-sendmail@latest
```

### Sendmail Running

```sh
cat email.txt | ./graph-sendmail
```

### Sendmail Command-Line Options

* `--clientid`: Client/Application ID (string)
* `--sentitems`: Save to senders sent items (bool)
* `--secret`: Client Secret (string)
* `--tenantid`: Tenant ID (string)
* `--quiet`: Silence any output (bool)
* `--debug`: Enable debug logging (bool)

All command line options may be specified as environment variables in the form of `SENDMAIL_<option>`.

## Status

### What works

Based on limited testing, sending of plain text and HTML emails with or without an attachment works correctly.

Sending to one or more recipients via Cc/Bcc also works.

### TODO

* Make logging better
* Message queueing (if this is even a good idea)
* SMTP authentication
* Allow running as a service on Windows
* Test with wider variety of devices
* Implement unit tests
