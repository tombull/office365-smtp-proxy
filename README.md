# graph-smtpd

**This is a work in progress and although it is functional, it may contain bugs. Use at your own risk**

The planned functionality of this service is a SMTP daemon that will accept emails via SMTP and then submit them via the Microsoft Graph API to Microsoft 365, however in theory the submission process could be generalised to support any API driven email submission service.

The idea is to allow this service to run locally and accept emails from devices/systems that cannot support the modern authentication requirements of Exchange Online, such as OAuth2, and then relay them securely to Microsoft 365.

## Azure AD/Entra ID Requirements

1. Create an "App Registration"
2. Note down the Client/Application ID and Tenant ID
3. Create a Client Secret and record the value
4. Set permissions (see below)

### Permissions

The App Registration requirs the `mail.send` Graph API permission and to have admin consent granted in your tenant.

This will allow the service to send as any users in your environment.

To limit this ability to specific mailboxes/senders, it is possible to implement a `ApplicationAccessPolicy` to control this as follows:

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
    -e SMTPD_CLIENTID="clientid" \
    -e SMTPD_SECRET="secret" \
    -e SMTPD_TENANTID="tenantid" \
    ghcr.io/andrewheberle/graph-smtpd:v0.1.1
```

### Command-Line Options

* `--addr`: Listen address (default = "localhost:2525")
* `--cert`: Certificate for enabling STARTTLS
* `--clientid`: Client/Application ID
* `--key`: Private key for enabling STARTTLS
* `--secret`: Client Secret
* `--senders`: Allowed senders
* `--sources`: Allowed source IP addresses
* `--tenantid`: Tenant ID

All command line options may be specified as environment variables in the form of `SMTPD_<option>`.

### Configuration File

All configuration options may be provided in a YAML or JSON configuration file that may be provided using the `--config` option or will be looked for in the current working directory as `config.<yaml|json>`.

## Status

### What works

Based on limited testing, sending of plain text emails with or without an attachment works correctly.

### Untested

* HTML emails
* Multiple recipients
* Multiple attachments
* CC/BCC

### TODO

* Make logging better
* Message queueing (if this is even a good idea)
* SMTP authentication
* Allow running as a service on Windows
