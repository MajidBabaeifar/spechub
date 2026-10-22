# Webhooks

**Status:** Draft · **Prefix:** HOOK

## Purpose

Send signed HTTP callbacks for events an app defines, reliably and safely.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| HOOK-1 | Subscription: `{id, url, events [names defined by the requiring spec], active (default true), secret (generated, ≥ 256 bits, shown once), scope (owner set by the requiring spec), template?}`. `(scope, url)` unique. | duplicate url in scope → refused |
| HOOK-2 | URL check on save and before each send: `https` only; host must not resolve to loopback, private, link-local or metadata addresses. | `https://169.254.169.254` → refused |
| HOOK-3 | Delivery: `POST`, JSON body `{id (delivery id), event, created_at, data}`, headers `Webhook-Id`, `Webhook-Timestamp` (unix s), `Webhook-Signature: v1,<base64 HMAC-SHA256(secret, id + "." + timestamp + "." + body)>`. | receiver verifies signature with secret |
| HOOK-4 | Redirects not followed; timeout 10 s; any non-2xx or timeout is a failure. | receiver 500 → retried |
| HOOK-5 | Retries with backoff: 30 s, 2 min, 10 min, 1 h, 6 h (6 attempts total); same delivery `id` on every attempt. | 500 then 200 → delivered once, same id |
| HOOK-6 | Each attempt logged `{delivery id, attempt, status code, duration, error?}`, kept 30 days, visible to the subscription owner; failed deliveries can be resent. | manual: log shows attempts |
| HOOK-7 | A subscription failing every delivery for 5 consecutive days is set inactive and its owner notified. | manual |
| HOOK-8 | Optional `template` renders the body from `data` with a logic-less template language (Mustache); result must be valid JSON, else save is refused. | invalid template → save refused |
| HOOK-9 | "Send test" posts a sample of a chosen event, returns the receiver's status, and is subject to HOOK-2. | test → status shown |
| HOOK-10 | Events are sent at least once and in no guaranteed order; receivers dedupe by `id`. | manual: documented for receivers |
| HOOK-11 | Time-based events (e.g. "starts in") are stored as scheduled deliveries and cancelled when the requiring spec says so. | manual: see requiring spec |

## Out of scope

- Zapier/Make-specific payloads, per-event custom headers.

## Decisions

- Signature scheme follows Standard Webhooks (id + timestamp in the signed content blocks replays); the source signed only the body.
- Non-2xx counts as failure and is retried; the source retried only network errors.
