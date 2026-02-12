# Akavelog Backend


## Getting Started

> Add instructions to set up locally under Getting Started section above!!!


## Step-By-Step Implementation Plan

> Note: GPT-5.2-Codex was used to generate the initial implementation plan below, which is manually verified by me!!!!

### Phase 1: Foundation

**Step 1: Project Setup**
- Initialize Go module with Echo framework
- Set up folder structure (`cmd/`, `internal/`, `pkg/`)
- Configure environment loading (`koanf`)
- Add structured logging (`zerolog`)

**Step 2: Database Layer**
- Set up PostgreSQL (local Docker)
- Configure `pgx` connection pool
- Write migrations with `tern`:
  - `projects` table
  - `api_keys` table
  - `log_batches` table (metadata index)
  - `alert_rules` + `alert_events` tables



### Phase 2: Ingestion Pipeline

**Step 3: Ingestion Endpoint**
- Create `POST /ingest` handler
- Validate request body (`timestamp`, `service`, `level`, `message`, `tags`)
- API key validation middleware → extract `project_id`

**Step 4: Batcher**
- In-memory buffer per project
- Flush on:
  - Batch size threshold (e.g., `1000` logs)
  - Time threshold (e.g., `30 seconds`)
- Compress batch with `gzip` / `zstd`


### Phase 3: Storage Layer

**Step 5: Akave O3 Integration**
- Initialize AWS SDK v2 with custom endpoint (`https://o3-rc2.akave.xyz`)
- Implement storage adapter:
  - `PutObject` → upload compressed batch
  - `GetObject` → retrieve batch
  - `HeadObject` → verify upload
- Return `o3_object_key` after upload


### Phase 4: Indexing

**Step 6: Metadata Index Write**
- After O3 upload succeeds:
  - Extract metadata: `project_id`, `service`, `ts_start`, `ts_end`, `levels`, `tags`
  - Insert into `log_batches` table with `o3_object_key`
- Ensure atomicity (upload + index write)


### Phase 5: Query Engine

**Step 7: Query Endpoint**
- Create `POST /query` handler
- Parse filters: time range, service, level, keyword

**Step 8: Metadata Lookup**
- Query `log_batches` table for matching `o3_object_key` values
- Filter by project, time, service, level

**Step 9: Batch Fetch + Filter**
- Fetch matching batches from Akave O3 via `GetObject`
- Decompress in memory
- Apply keyword/field filters
- Stream results to client (SSE or chunked JSON)


### Phase 6: Frontend

**Step 10: Next.js Setup**
- Initialize Next.js + TypeScript + TailwindCSS
- Set up API client for backend

**Step 11: Log Explorer UI**
- Time-range picker
- Service/level/tag dropdowns
- Keyword search input
- Streaming log list with infinite scroll
- Log detail panel


### Phase 7: Alerting 


**Step 12: Alert Rule CRUD**
- `POST /alerts` → create rule
- `GET /alerts` → list rules
- `DELETE /alerts/:id` → delete rule

**Step 13: Background Worker**
- Run every `60 seconds`
- Fetch enabled rules
- Execute query against metadata index
- Evaluate threshold conditions
- Record `alert_events` and trigger notifications


### Phase 8: Identity

**Step 14: Project + API Key Management**
- `POST /projects` → create project + generate API key
- Middleware validates `X-API-Key` on all requests
- Scope all queries to `project_id`

**Step 15: Production Hardening**
- Rate limiting
- Retry logic for O3 uploads
- Observability (Prometheus metrics, health check)
- Error handling + graceful shutdown




## Resources

**Project Layout** - https://github.com/golang-standards/project-layout

**echo framework** - "https://echo.labstack.com/docs/quick-start"

**pgx - SQL Driver** - https://github.com/jackc/pgx

**tern - SQL Migrator** - https://github.com/jackc/tern

**zerolog - JSON Logger** - https://github.com/rs/zerolog

**newrelic -Monitoring and Observability** - "https://pkg.go.dev/github.com/newrelic/go-agent/v3@v3.40.1/newrelic"

**validator** - https://github.com/go-playground/validator

**koanf - Configuration Management** - https://github.com/knadh/koanf

**testify - for testing** - https://github.com/stretchr/testify

**taskfile** - https://taskfile.dev/

**AsyncQ - queueing tasks and processing them asynchronously with workers** - https://github.com/hibiken/asynq