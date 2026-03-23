<p align="center">
  <a href="https://whynesspower.com/">
    <img src="https://github.com/user-attachments/assets/fc8df502-bdd2-4e20-a845-4f185113e3c9" width="150" alt="Open Context Logo">
  </a>
</p>

<h1 align="center">
Open Context: An open source platform for Context Graphs & Engineering for your AI agents
</h1>

<h2 align="center">Examples, Graph Engine, & More</h2>

<br />

<p align="center">
  <a href="https://twitter.com/intent/follow?screen_name=whyesspower" target="_new"><img alt="Twitter Follow" src="https://img.shields.io/twitter/follow/whynesspower"></a>
</p>

# Open Context

Open Context is a self-hosted context service for AI applications that want a **Zep Cloud-compatible** API without depending on a hosted control plane. It combines a Go API, PostgreSQL persistence, Graphiti-powered graph extraction, Neo4j storage, and a lightweight admin UI so you can ingest conversations, build relationship-aware memory, and query it through the familiar `/api/v2` surface.

## What It Does

Open Context is designed for teams that want to run agent memory and graph-backed context retrieval on their own infrastructure while keeping compatibility with existing Zep-style SDK flows.

### How it works

1. **Ingest messages and metadata** through the Zep-compatible API.
2. **Persist operational state** in PostgreSQL and forward graph-relevant data to Graphiti.
3. **Build temporal graph structure** in Neo4j for nodes, edges, episodes, and search.
4. **Retrieve context and graph data** from the same API contract used by existing clients.

## Quick Start

### Docker setup

1. Copy env:

```bash
cp .env.example .env
```

2. Set `OPENAI_API_KEY` and `OPEN_CONTEXT_API_KEY` in `.env`.

Before you start the stack, replace the sample defaults in `.env` for `OPEN_CONTEXT_API_KEY` and `OPEN_CONTEXT_ADMIN_*`. The example values are convenient for local development, but they should not be used outside a private machine.

3. Start stack:

```bash
docker compose --env-file .env up --build
```

### Services

- `http://localhost:8000` — Go API (`Authorization: Api-Key …`)
- `http://localhost:8003` — Graphiti
- `http://localhost:3000` — Admin UI (login from `OPEN_CONTEXT_ADMIN_*`)
- `http://localhost:8080` — Example FastAPI (`/docs`)

## SDK Configuration

`zep-python` reads `ZEP_API_URL` (host only) and `ZEP_API_KEY`. Point them at this backend:

```bash
export ZEP_API_URL=http://localhost:8000
export ZEP_API_KEY=your-open-context-api-key
```

Open Context’s own settings use the `OPEN_CONTEXT_*` names; compose maps `ZEP_API_KEY` from `OPEN_CONTEXT_API_KEY` for the example container. Use the same value you configured for `OPEN_CONTEXT_API_KEY` in `.env`.

## Repository Layout

- `backend/` — Go service implementing `/api/v2`
- `graphiti/` — vendored Graphiti + FastAPI graph service
- `sdks/zep-python/` — Zep Cloud Python SDK (unchanged contract)
- `frontend/` — Next.js admin + D3 graph view
- `examples/fastapi-app/` — minimal SDK demo

## Notes

- API path remains `/api/v2` to match the official SDKs (SDK major version != URL version).
- Legacy Zep CE (`zep/legacy`) is a reference for patterns only; feature coverage is driven by the current SDK surface.
- This monorepo vendors `graphiti/` and `sdks/zep-python/` without nested `.git/` metadata so everything can be tracked in one repository. If you re-copy those trees from upstream, remove their inner `.git` (or use submodules) before committing.

## Implementation Status

The Go `/api/v2` surface is implemented to be SDK-compatible for common flows, including users, threads and messages, graph search, nodes and edges listing, episode proxying, templates, instructions, and entity types.

### Completed SDK compatibility fixes

- `GET /users/{id}/threads` returns a bare JSON array (matches `List[Thread]` parse in SDK)
- `GET /tasks/{id}` returns `progress` as `{message, stage}` and `error` as `{code, details, message}` objects with all timestamp fields
- `GET /projects/info` nests response under `project` key with correct field names
- `GET /threads/{id}/context` respects `template_id` query param and wires custom/user-summary instructions into context
- `POST /threads/{id}/messages` respects `ignore_roles` to exclude roles from Graphiti ingestion
- `GET /threads` respects `order_by` and `asc` query params for sorting
- `GET /users-ordered` respects `search` query param for filtering by user_id, email, first/last name

### Pending architectural work

- [ ] **Entity types -> Graphiti ontology push**: `PUT /entity-types` saves custom entity and edge type definitions to PostgreSQL but does not push them to the Graphiti Python service. Graphiti processes entity extraction via LLM prompts; injecting custom entity types requires modifying the `/messages` ingest route in `graphiti/server/graph_service/routers/ingest.py` to look up entity type definitions and include them in the extraction prompt. This is a cross-service change that touches the Python service.
- [ ] **`POST /graph/patterns` -> LLM-based pattern detection**: the SDK's `graph.detect_patterns()` expects `DetectPatternsResponse` from an LLM-analyzed pattern scan. The current implementation returns statistical frequency counts (label distribution, edge name counts, highest-degree node). A full implementation needs an LLM call from the Go backend or Graphiti service to analyze the graph and return typed pattern objects matching the SDK schema.

## Commit Style

Use angular-style, lowercase messages, one small logical change per commit, for example `feat: add thread list handler`. The project targets a high commit count over its lifetime, so prefer many tiny commits over large squashes.
