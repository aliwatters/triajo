# Architecture

Dochore follows the same 3-tier pattern as travelblog: frontend components call Next.js API routes, which proxy to a Go API that owns all database access.

## System Diagram

```
                    ┌─────────────────────────────┐
                    │       Ingress (Nginx)        │
                    │   admin.dochore.test          │
                    │   api.dochore.test            │
                    │   dochore.com (public)        │
                    └──────┬──────────┬────────────┘
                           │          │
              ┌────────────▼──┐  ┌────▼──────────┐
              │  dochore-app   │  │  dochore-web   │
              │  (Next.js)    │  │  (Next.js)    │
              │  Admin UI     │  │  Public site  │
              │  Port 3000    │  │  Port 3010    │
              └──────┬────────┘  └───────────────┘
                     │ /api/*
              ┌──────▼────────┐
              │  dochore-api   │
              │  (Go + Gin)   │
              │  Port 8080    │
              └──────┬────────┘
                     │
              ┌──────▼────────┐
              │   MongoDB     │
              │   8.0         │
              │   Port 27017  │
              └───────────────┘
```

## Services

### dochore-app (Admin UI)

The primary interface. Next.js with App Router.

- **Framework**: Next.js (latest stable), React, TypeScript
- **Styling**: Tailwind CSS + Shadcn UI
- **Auth**: better-auth (Google OAuth initially, add more later)
- **Port**: 3000
- **Domain**: admin.dochore.test (dev), admin.dochore.com (future)

Key pages:
- `/` — Dashboard: today's tasks grouped by tag
- `/inbox` — Untagged tasks needing triage
- `/tasks` — Full task list with filters
- `/people` — Manage handlers (family, VA, housekeeper)
- `/rules` — Auto-triage rules (e.g., "dog" → [FAMILY])

### dochore-web (Public Site)

Static landing page. Does nothing functional in v1.

- **Framework**: Next.js (static export)
- **Port**: 3010
- **Domain**: dochore.com

### dochore-api (Go API)

All business logic and database access.

- **Framework**: Go + Gin
- **Database**: MongoDB via official Go driver
- **Port**: 8080
- **Domain**: api.dochore.test (dev)

Key endpoints:
```
POST   /v1/tasks          Create task
GET    /v1/tasks           List tasks (with filters)
GET    /v1/tasks/:id       Get task
PATCH  /v1/tasks/:id       Update task (status, tag, assignee)
DELETE /v1/tasks/:id       Delete task

POST   /v1/tasks/triage    Auto-triage untagged tasks
GET    /v1/handlers        List handlers (people/agents)
POST   /v1/handlers        Create handler

GET    /v1/health          Health check
```

### dochore-mcp (MCP Server)

Exposes dochore to AI agents via Model Context Protocol. Can be a thin wrapper around the Go API or a standalone Go binary.

Tools:
- `task_create` — Create a task with optional tag
- `task_list` — List tasks with filters (tag, status, handler, date range)
- `task_update` — Update task status, tag, or assignee
- `task_triage` — Run auto-triage on inbox
- `inbox_count` — Quick count of untagged tasks

## Database

MongoDB 8.0 on hyperion (192.168.4.106). Already running, accessible from 192.168.4.0/24.

### Collections

```
dochore.tasks
{
  _id: ObjectId,
  title: string,
  description: string,           // markdown
  tag: string,                   // "ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"
  status: string,                // "inbox", "pending", "in_progress", "completed", "cancelled"
  handler: string | null,        // who's doing it (name or agent ID)
  priority: string,              // "urgent", "high", "normal", "low"
  due: Date | null,
  source: string,                // "manual", "agent", "email", "calendar", "voice"
  metadata: object,              // flexible — source-specific data
  created_at: Date,
  updated_at: Date,
  completed_at: Date | null
}

dochore.handlers
{
  _id: ObjectId,
  name: string,                  // "Ali", "Mom", "VA - Sarah", "Claude"
  type: string,                  // "me", "family", "va", "housekeeper", "ai", "service"
  tags: [string],                // which tags this handler covers
  contact: {                     // how to reach them
    email: string,
    phone: string,
    agent_id: string             // for AI handlers
  },
  active: boolean,
  created_at: Date
}

dochore.rules
{
  _id: ObjectId,
  pattern: string,               // regex or keyword match on title/description
  tag: string,                   // auto-assign this tag
  handler: string | null,        // auto-assign this handler
  priority: string | null,       // auto-set priority
  active: boolean,
  created_at: Date
}
```

### Indexes

```
tasks: { status: 1, tag: 1 }
tasks: { handler: 1, status: 1 }
tasks: { due: 1 } (sparse)
tasks: { created_at: -1 }
rules: { active: 1 }
```

## Deployment

Docker Compose on hyperion. No Kubernetes — this is a household tool.

### docker-compose.yml

```yaml
services:
  dochore-app:
    build: ./src/dochore-app
    ports: ["3000:3000"]
    environment:
      - TRIAJO_API_URL=http://dochore-api:8080
    depends_on: [dochore-api]

  dochore-web:
    build: ./src/dochore-web
    ports: ["3010:3010"]

  dochore-api:
    build: ./src/dochore-api
    ports: ["8080:8080"]
    environment:
      - MONGO_URI=mongodb://host.docker.internal:27017
      - MONGO_DATABASE=dochore
    extra_hosts:
      - "host.docker.internal:host-gateway"

  web:
    image: nginx:alpine
    ports: ["80:80", "443:443"]
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
    depends_on: [dochore-app, dochore-web, dochore-api]
```

### DNS (local dev)

Add to `/etc/hosts` or dnsmasq:
```
127.0.0.1  admin.dochore.test
127.0.0.1  api.dochore.test
127.0.0.1  dochore.test
```

On hyperion (production):
```
192.168.4.106  admin.dochore.test
192.168.4.106  api.dochore.test
```

## Tech Stack Summary

| Layer | Technology |
|-------|-----------|
| Admin UI | Next.js, React, TypeScript, Tailwind, Shadcn |
| Public Site | Next.js (static) |
| API | Go 1.24, Gin |
| Database | MongoDB 8.0 |
| Auth | better-auth (Google OAuth) |
| MCP | Go binary (dochore-mcp) |
| Containers | Docker, Docker Compose |
| Reverse Proxy | Nginx |
| Host | Hyperion (192.168.4.106) |
| Network | Tailscale for remote access |
