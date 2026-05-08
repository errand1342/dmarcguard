# Azure App Registration Setup for Microsoft Graph

This guide walks through setting up Azure AD authentication for fetching DMARC reports from Exchange Online via Microsoft Graph API.

## Prerequisites

- Azure subscription with appropriate permissions
- Shared mailbox in Exchange Online where DMARC reports arrive
- PowerShell (optional, for Graph API setup scripts)

## Step 1: Create App Registration

1. Go to [Azure Portal](https://portal.azure.com) → **Azure AD** → **App registrations**
2. Click **+ New registration**
3. Configure:
   - **Name**: `parse-dmarc` (or your preference)
   - **Supported account types**: `Accounts in this organizational directory only`
   - **Redirect URI**: Leave empty (not needed for daemon/service account)
4. Click **Register**
5. Copy and save:
   - **Application (client) ID** — needed for config
   - **Directory (tenant) ID** — needed for config

## Step 2: Create Client Secret

1. In the app registration, go to **Certificates & secrets**
2. Click **+ New client secret**
3. Configure:
   - **Description**: `parse-dmarc-secret`
   - **Expires**: Choose based on your security policy (24 months recommended)
4. Click **Add**
5. **Immediately copy the secret value** (only shown once!)
   - Save as `GRAPH_CLIENT_SECRET` in your config

## Step 3: Grant API Permissions

1. In the app registration, go to **API permissions**
2. Click **+ Add a permission**
3. Select **Microsoft Graph**
4. Select **Application permissions** (not Delegated)
5. Search for and select:
   - `Mail.Read` — read email from mailbox
   - OR `Mail.ReadWrite` — read and mark emails as read (recommended)
6. Click **Add permissions**
7. Click **Grant admin consent for [your tenant]** (requires admin)
   - Status should show "✓ Granted"

## Step 4: Grant Mailbox Access

The app needs permission to access the shared mailbox. Use the Exchange Admin Center or PowerShell:

### Option A: PowerShell (Recommended)

```powershell
# Connect to Exchange Online
Connect-ExchangeOnline -Credential (Get-Credential)

# Get the app's service principal object ID
$appId = "YOUR_APP_CLIENT_ID"
$servicePrincipal = Get-ServicePrincipal | Where-Object { $_.AppId -eq $appId }

# Grant access to shared mailbox
Add-MailboxPermission -Identity "dmarc-reports@yourdomain.com" `
  -User $servicePrincipal.ObjectId `
  -AccessRights FullAccess `
  -InheritanceType All
```

### Option B: Azure Portal

1. Go to **Exchange Admin Center** → **Recipients** → **Mailboxes**
2. Select the shared mailbox
3. Click **Delegated mailbox permissions** or **Manage delegated access**
4. Add your app's service principal with **Read** or **Read, Delete** access

## Step 5: Configure parse-dmarc

Update your `config.json`:

```json
{
  "graph": {
    "enabled": true,
    "tenant_id": "YOUR_TENANT_ID",
    "client_id": "YOUR_APP_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "mailbox": "dmarc-reports@yourdomain.com",
    "folder_path": "INBOX",
    "mark_as_read": true
  }
}
```

Or set environment variables:

```bash
export GRAPH_ENABLED=true
export GRAPH_TENANT_ID=YOUR_TENANT_ID
export GRAPH_CLIENT_ID=YOUR_APP_CLIENT_ID
export GRAPH_CLIENT_SECRET=YOUR_CLIENT_SECRET
export GRAPH_MAILBOX=dmarc-reports@yourdomain.com
export GRAPH_FOLDER_PATH=INBOX
export GRAPH_MARK_AS_READ=true
```

## Step 6: Verify Permissions

Test with a simple script to confirm the app can access the mailbox:

```bash
curl -X POST "https://login.microsoftonline.com/YOUR_TENANT_ID/oauth2/v2.0/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=YOUR_APP_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET&scope=https://graph.microsoft.com/.default&grant_type=client_credentials"
```

This should return an access token. Then test mailbox access:

```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  "https://graph.microsoft.com/v1.0/users/dmarc-reports@yourdomain.com/mailFolders/inbox/messages?$top=1"
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `AADSTS700016: Application with identifier 'X' not found` | Verify `GRAPH_CLIENT_ID` is correct |
| `AADSTS7000215: Invalid client secret provided` | Secret has expired or is incorrect; regenerate in Azure Portal |
| `AADSTS65001: User or admin has not consented` | Grant admin consent for API permissions (Step 3, final step) |
| `Access Denied to mailbox` | Check mailbox permissions are granted to the service principal (Step 4) |
| `ERR_GRAPH_NOT_CONFIGURED` | Graph is not enabled; set `GRAPH_ENABLED=true` |

## Alternative: Certificate Authentication

For production, certificate-based auth is more secure. To use a certificate instead of a client secret:

1. Generate a self-signed certificate:
   ```bash
   openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes
   ```

2. Upload to Azure:
   - Go to **App Registration** → **Certificates & secrets** → **Certificates**
   - Click **Upload certificate**
   - Select your `cert.pem`

3. Update config:
   ```json
   {
     "graph": {
       "cert_path": "/path/to/cert.pem",
       "cert_key_path": "/path/to/key.pem"
     }
   }
   ```

(Certificate support coming in v2.0)

## References

- [Microsoft Graph Mail API](https://learn.microsoft.com/en-us/graph/api/resources/message?view=graph-rest-1.0)
- [Azure App Registration](https://learn.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app)
- [Microsoft Identity Platform](https://learn.microsoft.com/en-us/azure/active-directory/develop/)
