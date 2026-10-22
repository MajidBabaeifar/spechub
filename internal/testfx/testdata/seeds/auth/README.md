---
specfork: 1
name: "@specfork/auth"
summary: Email/password, OIDC and magic-link sign-in, TOTP 2FA, password reset, sessions, API keys.
license: CC-BY-4.0
derived_from:
  repo: https://github.com/calcom/cal.com
  commit: 0f19cc46ef6b6426ee398c670575b3c96be7af9e
  url: https://cal.com
  license: MIT
---

# Auth

Accounts and sign-in for a multi-user web app. Partly derived from Cal.com's auth; password and rate-limit rules follow NIST 800-63B instead.

| Area | Prefix | Covers |
|---|---|---|
| [auth.md](specs/auth.md) | AUTH | Users, sign-up, sign-in, 2FA, reset, sessions, API keys, rate limits, deletion |
