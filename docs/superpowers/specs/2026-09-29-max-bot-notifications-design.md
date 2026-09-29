# MAX Bot Notifications Design

## Goal

Turn the existing `/start` MAX bot integration into a real notification channel for residents and organization staff, with links back to the relevant request in the mini app.

## Scope

- Keep the existing `/start`, `bot_started`, and mini-app button behavior.
- Link a signed MAX mini-app identity to the currently authenticated application account.
- Notify the request owner about public organization replies, status changes, assignment/visit changes, and resolution results.
- Notify managers of the primary and contractor organizations about new requests and new public resident messages.
- Never send internal notes through MAX.
- Include a short event summary and a button that opens the request card in the mini app.
- Keep the existing long-polling update consumer. Production webhook migration is outside this change.

## Architecture

### Account linking

The frontend already receives `window.WebApp.initData`. After the user has an authenticated application session, it sends that raw string to a new authenticated backend endpoint. The backend validates the MAX HMAC-SHA256 signature with `MAX_BOT_TOKEN`, rejects stale or malformed data, extracts the MAX user and dialog identifiers, and upserts the link for the current application user.

Only validated server-side data is stored. `initDataUnsafe` is not trusted for authentication. Opening the site in a normal browser does not attempt linking.

### Storage and recipients

A migration adds a one-to-one link from an application user to a MAX user and dialog. Repository queries resolve:

- the request owner's MAX destination;
- MAX destinations of managers belonging to the request's primary or contractor organization.

Duplicate destinations are removed before sending. Missing links are skipped without failing the business action.

### Notification delivery

The existing notification abstraction is replaced with a real MAX notifier when `MAX_BOT_TOKEN` is configured and retains the logging adapter otherwise. Notification methods receive a typed event rather than a preformatted status string, resolve recipients through the repository, format Russian text, and add a MAX mini-app deep link with a request-specific start parameter.

Delivery remains best-effort in this change: MAX errors are logged and do not roll back a request, message, or status transition. Durable retries/outbox delivery are outside scope.

## Event rules

| Event | Resident | Organization managers |
| --- | --- | --- |
| New request | Confirmation | Primary and contractor |
| Public resident message | No self-notification | Primary and contractor |
| Public organization reply / info request | Request owner | No self-notification |
| Internal note | Nobody | Nobody |
| Status change | Request owner | No notification |
| Assignment, visit, manual route, resolution decision | Request owner | No notification |

Notification text contains the request number, human-readable event/status, and a short sanitized excerpt when applicable. The button opens the request card; authorization in the mini app remains the final access control.

## Security and validation

- Verify MAX `initData` exactly according to the documented sorted-parameter HMAC-SHA256 algorithm.
- Require one `hash`, one valid user object, a recent `auth_date`, and a dialog chat before linking.
- Compare signatures in constant time.
- Never expose or log the bot token or raw `initData`.
- Keep all existing request access checks; the notification deep link does not grant access.

## Testing

- Unit tests for valid, invalid, duplicate, and stale MAX launch data.
- Repository/SQL coverage for linking and recipient lookup where existing database tests support it.
- Service tests for resident/manager routing, deduplication, internal-note exclusion, and graceful missing-link behavior.
- MAX client tests for notification payload and request-specific mini-app button.
- Frontend bridge test for one-time linking only inside MAX after application authentication.
- Run backend Go tests and the focused frontend test suite.
