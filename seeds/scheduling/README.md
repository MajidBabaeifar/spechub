---
spechub: 1
name: "@spechub/scheduling"
summary: Booking pages with availability, team round robin, calendar sync, payments and webhooks.
license: CC-BY-4.0
requires:
  "@spechub/auth": "^0.1"
  "@spechub/teams": "^0.1"
  "@spechub/webhooks": "^0.1"
derived_from:
  repo: https://github.com/calcom/cal.com
  commit: 0f19cc46ef6b6426ee398c670575b3c96be7af9e
  url: https://cal.com
  license: MIT
---

# Scheduling

People and teams publish booking pages; others book time without an account.
Behaviour derived from Cal.com's open-source code (Cal.diy, MIT), written fresh and stack-free.

| Area | Prefix | Covers |
|---|---|---|
| [event-types.md](specs/event-types.md) | EVT | Bookable templates: durations, notice, window, limits, seats, confirmation |
| [schedules.md](specs/schedules.md) | AVL | Weekly hours, overrides, time zones, time off, holidays |
| [slots.md](specs/slots.md) | SLT | Which start times are offered, buffers, limits, holds |
| [bookings.md](specs/bookings.md) | BKG | Create, confirm, reschedule, cancel, seats, recurring, no-show |
| [team-events.md](specs/team-events.md) | RR | Collective and round-robin team events, fairness, team permissions |
| [calendars.md](specs/calendars.md) | CAL | Calendar and video connections, busy times, writing events |
| [payments.md](specs/payments.md) | PAY | Paid events, timeout, refunds, no-show fees |
| [notifications.md](specs/notifications.md) | NTF | Emails and webhook events |
| [public-pages.md](specs/public-pages.md) | PUB | Profile, booking flow, manage booking, embeds |

All times are stored in UTC; all durations are whole minutes.
