# Schema

MongoDB 8.0. Database: `ginla`. Four ginla collections + better-auth managed collections.

```
┌─────────────────────────────────────────────────────────────────┐
│                        ginla database                           │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │  households   │───▶│   handlers   │    │    rules     │      │
│  │              │    │              │    │              │      │
│  │  members[]   │    │  type        │    │  pattern     │      │
│  │  ∟ user_id   │    │  tags[]      │    │  → tag       │      │
│  │  ∟ role      │    │  contact{}   │    │  → handler   │      │
│  │  ∟ handler_id│───▶│              │    │  → priority  │      │
│  │  invites[]   │    │              │    │              │      │
│  │  ∟ email     │    │              │    │              │      │
│  │  ∟ token     │    │              │    │              │      │
│  └──────┬───────┘    └──────▲───────┘    └──────────────┘      │
│         │                   │                                   │
│         │ household_id      │ handler_id                        │
│         │                   │                                   │
│  ┌──────▼───────────────────┴──────────────────────────┐       │
│  │                      tasks                           │       │
│  │                                                      │       │
│  │  title, description                                  │       │
│  │  tag ─────── ME | AI | VA | FAMILY | HOUSEKEEPER     │       │
│  │  status ──── inbox | pending | active | done | cancel│       │
│  │  priority ── urgent | high | normal | low            │       │
│  │  source ──── manual | agent | email | calendar | ... │       │
│  │                                                      │       │
│  │  parent_id ──▶ tasks (self-ref for subtasks)         │       │
│  │  position ──── Number (manual sort order)             │       │
│  │  checklist[] ── [{text, done}] (inline checkboxes)   │       │
│  │  recurrence ── {rrule, next_at} (repeating tasks)    │       │
│  │  attachments[] ── [{url, name, type}]                │       │
│  │  activity[] ── [{action, by, at}] (audit trail)      │       │
│  │  meta {} ── (source-specific, freeform)              │       │
│  └──────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

## Collections

### households

The tenant. Everything is scoped to a household.

```js
{
  _id: ObjectId,
  name: "The Watters Household",
  members: [
    {
      user_id: "google-oauth|123",   // from better-auth
      name: "Ali",
      role: "owner",                 // "owner" | "admin" | "member" | "viewer"
      handler_id: ObjectId           // links to their handler record
    }
  ],
  invites: [
    {
      email: "jenny@example.com",
      role: "member",              // role they'll get on acceptance
      token: "inv_abc123def456",   // unique, used in invite link
      invited_by: "google-oauth|123",
      created_at: Date,
      expires_at: Date             // 7 days default
    }
  ],
  created_at: Date
}
```

- `members` is embedded — households are small (2-10 people), no need for a join table
- `user_id` comes from better-auth, opaque string
- `role` controls what they can do in the UI
- `handler_id` links a logged-in user to their handler identity for task assignment
- `invites` is embedded — transient (accepted or expired), small cardinality. On acceptance: remove from `invites[]`, add to `members[]`, create handler record

### tasks

The core object. Everything flows through here.

```js
{
  _id: ObjectId,
  household_id: ObjectId,

  // what
  title: "File Oregon annual report",
  description: "Registry #240894998, due May 14. $100 at sos.oregon.gov",
  checklist: [                       // inline checkboxes, no separate collection
    { text: "Look up business details", done: true },
    { text: "Fill form on SOS site", done: false },
    { text: "Pay $100", done: false }
  ],

  // who
  tag: "AI",                         // handler type: ME | AI | VA | FAMILY | HOUSEKEEPER | DELEGATE
  handler_id: ObjectId | null,       // specific handler assigned

  // when
  status: "active",                  // inbox → pending → active → done | cancelled
  priority: "normal",                // urgent | high | normal | low
  position: 0,                      // manual sort order (drag-and-drop), lower = higher in list
  due: Date | null,

  // where it came from
  source: "agent",                   // manual | agent | email | calendar | voice | screenshot
  meta: {                            // freeform, source-specific
    agent_session: "claude-2026-03-22",
    original_email_id: "196cf92afe25f308"
  },

  // structure
  parent_id: ObjectId | null,        // subtask of another task (self-referencing)

  // recurrence
  recurrence: {                      // null for one-off tasks
    rrule: "FREQ=YEARLY;BYMONTH=5;BYMONTHDAY=14",  // iCal RRULE
    next_at: Date                    // when to spawn next instance
  },

  // proof
  attachments: [
    { url: "https://...", name: "receipt.pdf", type: "application/pdf" }
  ],

  // audit trail (embedded, append-only)
  activity: [
    { action: "created", by: "ali", at: Date },
    { action: "tagged", by: "auto-triage", at: Date, detail: "matched rule: oregon → AI" },
    { action: "claimed", by: "claude-worker-3", at: Date },
    { action: "done", by: "claude-worker-3", at: Date }
  ],

  created_at: Date,
  updated_at: Date,
  done_at: Date | null
}
```

**Design notes:**
- `checklist` is embedded for lightweight checkboxes (grocery list, filing steps). Use `parent_id` for real subtasks that need their own lifecycle
- `activity` is append-only and embedded — keeps the full story with the task, no joins. Will grow but household tasks don't get thousands of updates
- `recurrence` uses standard iCal RRULE format — well-understood, parseable by every language
- `meta` is the escape hatch — anything source-specific goes here without polluting the schema
- `status` is simplified from 5 to 5 but renamed: `in_progress` → `active`, `completed` → `done` (shorter, matches how people talk)
- `position` enables manual drag-and-drop ordering in the UI. Nullable — when null, sort falls back to priority + created_at

### handlers

Who can do work. Broader than users — includes AI agents and services that never log in.

```js
{
  _id: ObjectId,
  household_id: ObjectId,
  name: "Claude",
  type: "ai",                       // me | family | va | housekeeper | ai | service
  tags: ["AI"],                      // which task tags this handler covers
  contact: {
    email: "ali.watters@gmail.com",  // for humans
    phone: "720-226-7602",           // for humans
    agent_id: "claude-worker-3"      // for AI handlers — maps to swarm agent
  },
  active: true,
  created_at: Date
}
```

### rules

Pattern-matching for auto-triage. When a task hits the inbox, rules run in order.

```js
{
  _id: ObjectId,
  household_id: ObjectId,
  name: "Oregon filings → AI",      // human label
  pattern: "oregon|annual report|secretary of state",  // regex on title + description
  tag: "AI",                         // auto-set tag
  handler_id: ObjectId | null,       // auto-assign handler
  priority: "normal",                // auto-set priority
  order: 10,                         // rules run lowest-first
  active: true,
  created_at: Date
}
```

## Indexes

```js
// tasks — primary query patterns
{ household_id: 1, status: 1, tag: 1 }         // "show me active AI tasks"
{ household_id: 1, handler_id: 1, status: 1 }  // "what's assigned to Claude?"
{ household_id: 1, due: 1 }                    // "what's due soon?" (sparse)
{ household_id: 1, created_at: -1 }            // "recent tasks"
{ household_id: 1, status: 1, position: 1 }   // "manual sort within a view"
{ household_id: 1, parent_id: 1 }              // "subtasks of X"
{ "recurrence.next_at": 1 }                    // cron: "which recurring tasks need spawning?"

// handlers
{ household_id: 1, type: 1, active: 1 }

// rules
{ household_id: 1, active: 1, order: 1 }
```

## Task Lifecycle

```
  ┌─────────┐
  │  inbox   │  ← untagged, needs triage (manual or auto-rule)
  └────┬─────┘
       │ triage (assign tag + optional handler)
  ┌────▼─────┐
  │ pending   │  ← tagged, waiting for handler to pick up
  └────┬─────┘
       │ claim / start
  ┌────▼─────┐
  │  active   │  ← being worked on
  └────┬─────┘
       │
  ┌────▼─────┐     ┌───────────┐
  │   done    │     │ cancelled  │  ← from any state
  └──────────┘     └───────────┘
```

## Recurrence Model

Recurring tasks use a **template + instance** pattern:

1. A task with `recurrence.rrule` is the **template**
2. A cron job checks `recurrence.next_at <= now`
3. It clones the template into a new task (no `recurrence` field) with `parent_id` pointing back to the template
4. It advances `next_at` on the template per the RRULE

This keeps recurring tasks as regular tasks in every query — no special cases in the UI or API.

## Multi-Tenancy

Every document has `household_id`. Every query includes it. No cross-household data leakage by construction. The compound indexes all lead with `household_id` so queries are scoped efficiently.

Users can only belong to one household for v1. The `households.members` array is the source of truth for access control.

## better-auth Collections (externally managed)

better-auth creates and manages its own collections in the same database. We don't define or migrate these — better-auth owns them. We only read `user_id` from them.

```
ginla.users          — managed by better-auth (user profiles, email, name)
ginla.sessions       — managed by better-auth (active sessions)
ginla.accounts       — managed by better-auth (OAuth provider links)
ginla.verifications  — managed by better-auth (email verification tokens)
```

The `households.members[].user_id` references `users._id` from better-auth. This is the only cross-reference between ginla collections and better-auth collections.
