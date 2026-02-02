# Storage Contract Method Indexer: Storage.sol-Aware Views for Akave Explorer

## Context / Background
Akave currently provides a blockchain explorer at **explorer.akave.ai**, built on **Blockscout**, which displays contract activity primarily in generic blockchain terms (transactions, logs, blocks).

For Akave’s storage use cases, users need a higher-level view that represents **Storage.sol contract interactions** in domain language (uploads, updates, deletions, etc.), rather than raw transactions.

### Existing Infrastructure
- **Explorer:** explorer.akave.ai (Blockscout-based)
- **Public RPC:** https://c1-us.akave.ai/ext/bc/56g16Hr1SHQRzdM8JLm3GKYv7APVHY8T2TyeZLvDVzCaTRS7W/rpc
- **Faucet:** faucet.akave.ai

## Problem Statement
Blockscout is excellent for generic blockchain exploration, but it does not provide a storage-specific representation of contract activity. Storage actions are currently visible only as:
- raw transactions
- method selectors / input data
- logs/events without a storage-aware “story”

This makes it hard for users and developers to:
- understand Storage.sol usage patterns
- find “all uploads” or “all updates” without manual decoding
- build analytics and dashboards around storage primitives

## Objective
Build an **indexer + API (and optionally UI)** that:
1. Uses the **public RPC** to read Storage.sol contract activity.
2. Decodes transactions/events into **method-centric records**.
3. Lists actions as **Storage methods** (e.g., “Upload”, “Register”, “Delete”, etc.) instead of generic “txns”.
4. Produces storage-aware views that are easy to query, filter, and consume.

This validates a path to augment (not replace) the existing Blockscout explorer with a storage-domain index.

## Core Idea
Users should be able to navigate storage contract activity as:
- **Methods** (contract function calls)
- **Events** (decoded log events)
- **Storage records** (domain-meaningful objects derived from method + event data)

Instead of:  
> “Transaction 0xabc… called contract 0xdef…”

They should see:  
> “Storage.uploadFile() called by 0x…, CID=…, size=…, bucket=…, status=…”

## Scope

### In Scope (MVP)
- Identify target Storage.sol contract address(es) and ABI required for decoding
- Indexer service that:
  - Pulls blocks and transactions via RPC
  - Filters to Storage.sol transactions and related logs
  - Decodes:
    - method name + inputs
    - emitted events + topics/data
  - Writes normalized records into a database
- API service that exposes:
  - list actions by method
  - filter by contract, caller, date range, bucket/file identifiers (where available)
  - fetch detail view for a single storage action
- Documentation + reproducible setup (local + testnet)

### Out of Scope (Initial)
- Full replacement of Blockscout UI
- Deep historical backfill optimizations beyond what’s required for MVP
- Advanced analytics dashboards (can be milestone 2/3)

## Intended Users / ICP
- Developers integrating storage contracts
- Protocol contributors debugging storage flows
- Operators monitoring storage usage
- Ecosystem teams building analytics and product surfaces on top of storage actions

## High-Level Architecture

### Indexer (Go recommended)
- Connects to Akave public RPC
- Tracks:
  - latest indexed block
  - reorg-safe indexing window (configurable)
- Filters:
  - tx.to == storage contract address
  - or logs emitted by storage contract
- Decodes:
  - function selectors to method names (via ABI)
  - input parameters
  - events into typed structures

### Storage / Database
- Postgres (preferred) or SQLite for MVP
- Core tables:
  - contracts
  - methods
  - events
  - actions (normalized storage-centric records)
  - indexing_state

### API Layer
- REST endpoints (or GraphQL)
- Examples:
  - `GET /actions?method=upload&fromBlock=...`
  - `GET /actions/{id}`
  - `GET /methods`
  - `GET /contracts`

### Optional UI (Phase 2)
- A lightweight “Storage Explorer” UI that:
  - mirrors explorer workflows but is method-first
  - links out to Blockscout transaction pages for provenance

## Expected Deliverables
- Indexer service (configurable contract list + ABI)
- DB schema + migrations
- API service and documentation
- Sample deployment guide (local + pointing to public RPC)
- Example dataset / screenshots demonstrating method-centric views
- Testing plan and basic automated tests

## Success Criteria
- Correctly decodes and indexes Storage.sol method calls and emitted events
- Users can list actions by method (not transaction) with useful filters
- Output is stable and reproducible, with clear docs for contributors
- Demonstrates clear value complementing existing Blockscout explorer

## Iterative Development Plan (Explicitly Designed for Refinement)
This proposal is intended to evolve as we learn from the contract ABI and real chain data.

Initial milestones should be structured to allow iteration:
1. **Contract discovery & ABI confirmation**
2. **Decode one method end-to-end** (method + key event)
3. **Expand coverage to N methods**
4. **Normalize into domain records**
5. **Expose API queries**
6. **(Optional) Add UI view + links to Blockscout**

As we proceed, we will refine:
- which methods are “primary”
- what fields are essential for users
- how to normalize method/event combinations into a single “action” record
- performance/backfill requirements

## References
- **Akave Explorer (Blockscout):** explorer.akave.ai
- **Public RPC:** https://c1-us.akave.ai/ext/bc/56g16Hr1SHQRzdM8JLm3GKYv7APVHY8T2TyeZLvDVzCaTRS7W/rpc
- **Faucet:** faucet.akave.ai
