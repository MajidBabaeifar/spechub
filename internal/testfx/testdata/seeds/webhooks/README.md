---
spechub: 1
name: "@spechub/webhooks"
summary: Signed, retried, SSRF-safe outbound webhooks for events the requiring spec defines.
license: CC-BY-4.0
derived_from:
  repo: https://github.com/calcom/cal.com
  commit: 0f19cc46ef6b6426ee398c670575b3c96be7af9e
  url: https://cal.com
  license: MIT
---

# Webhooks

Outbound HTTP callbacks. The requiring spec names the events and scopes; this spec delivers them.
Signing follows the Standard Webhooks scheme.

| Area | Prefix | Covers |
|---|---|---|
| [webhooks.md](specs/webhooks.md) | HOOK | Subscriptions, URL safety, signing, retries, logs, tests |
