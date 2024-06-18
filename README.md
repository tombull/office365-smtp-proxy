# graph-smtpd

**This is a work in progress and not yet fully functional**

The planned functionality of this service is a SMTP daemon that will accept emails via SMTP and then submit them via the Microsoft Graph API to Microsoft 365, however in theory the submission process could be generalised to support any API driven email submission service.

The idea is to allow this service to run locally and accept emails from devices/systems that cannot support the modern authentication requirements of Exchange Online, such as OAuth2, and then relay them securely to Microsoft 365.

## Azure AD/Entra ID Requirements

1. Create an "App Registration"
2. Note down the Client/Application ID and Tenant ID
3. Create a Client Secret and record the value
4. Grant the application "mail.send" API permission
5. Grant Admin Consent for the API permissions

## Running

### Command-Line Options

* `--addr`: Listen address (default = "localhost:2525")
* `--cert`: Certificate for enabling TLS
* `--clientid`: Client/Application ID
* `--key`: Private key for enabling TLS
* `--secret`: Client Secret
* `--tenantid`: Tenant ID

## Status

### What works

Based on limited testing, sending of plain text emails with or without an attachment works correctly.

### Untested

* HTML emails
* Multiple recipients
* Multiple attachments
* CC/BCC

### TODO

* Logging
* Message queueing (if this is even a good idea)
* SMTP authentication
* Allow running as a service on Windows
* Access/Relay controls
