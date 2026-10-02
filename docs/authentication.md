# Account authentication

New users (including a freshly bootstrapped admin) must change the initial password, then enroll an authenticator before using the application. Administrator password edits and resets require another password change. MFA resets require re-enrollment. Users cannot disable MFA themselves.

The server enforces onboarding for HTTP APIs, viewer WebSockets, relay viewers, and agent downloads authenticated by a user. Agent API-key authentication remains a separate machine credential. Keep that key private; it grants administrator-level access.

Passwords must contain 15–128 Unicode characters. Spaces are supported and preserved. A local common-password blocklist and repeated-character check reject obvious choices. No periodic expiry or mandatory character-class rules are imposed.

New passwords are screened against HIBP Pwned Passwords when creating accounts, editing an account password, resetting a password, or changing your own password. Outbound HTTPS to `api.pwnedpasswords.com:443` is required. Only the first five hexadecimal characters of the SHA-1 password digest leave the server; the full digest, password, and username are not sent. SHA-1 is used only for the range lookup, never for password storage. Responses request padding; zero-count padding records are ignored. Redirects, malformed/oversized responses, and timeouts fail the check. New-password operations return 503 without changing account data if screening is unavailable. No result means not found in that dataset, not proof that a password is safe.

Existing passwords are screened after successful password verification at login. A match sets the existing mandatory-change flag and records `password_compromised` in the authentication log. An unavailable lookup records `password_screen_unavailable`, does not clear an existing mandatory-change flag, and does not alone block login. Incorrect passwords are never queried. There is no background or bulk check of stored PBKDF2 hashes: checking requires the password supplied at login. Already-open remote sessions are not forcibly closed; changed account state invalidates existing tokens on their next validation. The bootstrap administrator must complete password-change onboarding, including screening, before normal use.

Passwords use PBKDF2-HMAC-SHA256, 600,000 iterations, a random 16-byte salt, and a 32-byte derived key. Existing SHA-256 hashes are verified and upgraded on successful password authentication; the original hash is never recoverable as a password. Existing passwords that fail the policy require a change after authentication.

Session tokens use HMAC-SHA256 and bind to current account credentials, MFA state, role, and branch. Password changes, MFA resets, and account changes invalidate old tokens and remembered-device credentials. Enrollment and MFA challenge tokens last 15 minutes; normal sessions last 24 hours. A remembered browser skips the MFA prompt for up to 14 days after successful MFA, while still requiring the password.

Deployment runs additive SQLite migrations. Back up the database before updating. Existing sessions and remembered-browser cookies from the previous token format will require login again. Legacy password hashes upgrade gradually as users log in. Existing active remote connections are not forcibly disconnected by this change; token checks apply when establishing new connections and making API requests.

Validation: `go test ./...`, `go vet ./...`, and `node --check web/static/app.js`. Authentication regression tests cover onboarding, direct API/WebSocket/relay/download restrictions, password hashing/migration, MFA verification, credential resets, and password policy across management routes.
