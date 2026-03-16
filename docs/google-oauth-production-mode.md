# Google OAuth: Moving from Testing to Production Mode

## Problem

Google OAuth apps in **Testing** mode have a critical limitation:
**refresh tokens expire after 7 days**. This means users must
re-authenticate weekly, which is unacceptable for a long-running server.

## Fix (5 minutes, no code changes)

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Select your project
3. Navigate to **APIs & Services** > **OAuth consent screen**
4. Check the **Publishing status** section
5. If it says **"Testing"**, click **"Publish App"**
6. For personal-use apps (fewer than 100 users), no verification is needed
7. After publishing, re-run `pkb auth` to get a new refresh token

## Why This Works

- **Testing mode**: Refresh tokens expire after 7 days
- **Production mode**: Refresh tokens do not expire (unless revoked or
  unused for 6 months)

After publishing, the new refresh token obtained via `pkb auth` will
persist indefinitely (as long as it is used at least once every 6 months).

## After Publishing

```bash
# Re-authenticate to get a non-expiring refresh token
pkb auth

# Verify the token works
make token-health

# Restart the production service
make deploy
```

## Troubleshooting

- **"This app isn't verified" warning**: Expected for personal-use apps.
  Click "Continue" during the OAuth flow. This warning only appears during
  authentication, not during normal API usage.
- **Token still expiring**: Ensure you re-ran `pkb auth` *after* publishing
  the app. Old tokens issued in Testing mode will still expire.
