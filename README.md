# open-context

<img width="1024" height="1024" alt="image" src="https://github.com/user-attachments/assets/14ead1b9-6119-4949-a1a5-d0dc5a6c80fb" />

Self-hosted **Zep Cloud–compatible** API (`/api/v2`) backed by **PostgreSQL**, **Graphiti**, and **Neo4j**, with an **admin UI** and a **FastAPI + zep-python** example.

## quick start (docker)

1. Copy env:

```bash
cp .env.example .env
```

2. Set `OPENAI_API_KEY` and `OPEN_CONTEXT_API_KEY` in `.env`.

3. Start stack:

```bash
docker compose --env-file .env up --build
```

Services:

- `http://localhost:8000` — Go API (`Authorization: Api-Key …`)
- `http://localhost:8003` — Graphiti
- `http://localhost:3000` — Admin UI (login from `OPEN_CONTEXT_ADMIN_*`)
- `http://localhost:8080` — Example FastAPI (`/docs`)

## sdk configuration

`zep-python` reads `ZEP_API_URL` (host only) and `ZEP_API_KEY`. Point them at this backend:

```bash
export ZEP_API_URL=http://localhost:8000
export ZEP_API_KEY=changeme
```

Open Context’s own settings use the `OPEN_CONTEXT_*` names; compose maps `ZEP_API_KEY` from `OPEN_CONTEXT_API_KEY` for the example container.

## repo layout

- `backend/` — Go service implementing `/api/v2`
- `graphiti/` — vendored Graphiti + FastAPI graph service
- `sdks/zep-python/` — Zep Cloud Python SDK (unchanged contract)
- `frontend/` — Next.js admin + D3 graph view
- `examples/fastapi-app/` — minimal SDK demo

## commit style

Use **angular-style**, **lowercase** messages, one small logical change per commit, e.g. `feat: add thread list handler`. The project targets a **high commit count** (on the order of **~500** over its lifetime); prefer many tiny commits over large squashes.

## notes

- API path remains `/api/v2` to match the official SDKs (SDK major version ≠ URL version).
- Legacy Zep CE (`zep/legacy`) is a **reference** for patterns only; feature coverage is driven by the current SDK surface.
- This monorepo vendors `graphiti/` and `sdks/zep-python/` **without nested `.git/` metadata** so everything can be tracked in one repository. If you re-copy those trees from upstream, remove their inner `.git` (or use submodules) before committing.

## implementation status

The Go `/api/v2` surface is implemented to be **SDK-compatible** for common flows (users, threads/messages, graph search, nodes/edges listing, episodes proxy, templates, instructions, entity types).

**Completed SDK compatibility fixes:**
- `GET /users/{id}/threads` returns a bare JSON array (matches `List[Thread]` parse in SDK)
- `GET /tasks/{id}` returns `progress` as `{message, stage}` and `error` as `{code, details, message}` objects with all timestamp fields
- `GET /projects/info` nests response under `project` key with correct field names
- `GET /threads/{id}/context` respects `template_id` query param and wires custom/user-summary instructions into context
- `POST /threads/{id}/messages` respects `ignore_roles` to exclude roles from Graphiti ingestion
- `GET /threads` respects `order_by` and `asc` query params for sorting
- `GET /users-ordered` respects `search` query param for filtering by user_id, email, first/last name

**Pending — requires architectural changes:**

- [ ] **Entity types → Graphiti ontology push**: `PUT /entity-types` saves custom entity/edge type definitions to Postgres but does not push them to the Graphiti Python service. Graphiti processes entity extraction via LLM prompts; injecting custom entity types requires modifying the `/messages` ingest route in `graphiti/server/graph_service/routers/ingest.py` to look up entity type definitions and include them in the extraction prompt. This is a cross-service change that touches the Python service.

- [ ] **`POST /graph/patterns` — LLM-based pattern detection**: The SDK's `graph.detect_patterns()` expects `DetectPatternsResponse` from an LLM-analyzed pattern scan. The current implementation returns statistical frequency counts (label distribution, edge name counts, highest-degree node). A full implementation needs an LLM call from the Go backend (or Graphiti service) to analyze the graph and return typed pattern objects matching the SDK's schema.
