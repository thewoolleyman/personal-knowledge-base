## Why

The dev server and production server both use port 8080, causing conflicts when both are running. The production server needs its own port so dev and production can coexist, especially when accessed via Tailscale from other clients.

## What Changes

- Change production server port from 8080 to 9000
- Dev server remains on port 8080
- Verify CI passes after change
- Verify production works from a remote Tailscale client before closing

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `infrastructure`: Production server port configuration changes from 8080 to 9000

## Impact

- Server configuration / startup code
- Any hardcoded port references
- Deployment configuration
- Documentation referencing the production port
