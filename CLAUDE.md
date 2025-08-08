# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview
PixelGW is a self-hosted WebSocket gateway for managing Pixlet apps and serving them to low-resolution pixel displays. It provides a REST API backend (Go) and React frontend for managing channels, devices, and applets.

## Development Commands

### Building and Deployment
```bash
# Generate API bindings from OpenAPI spec (required after modifying pixelgw.yaml)
make generate

# Build Docker image with frontend assets
make build

# Deploy development environment (port 8081, with HMR)
make deploy_test

# Deploy production environment (port 8080)
make deploy_prod
```

### Frontend Development
```bash
# Navigate to web directory first
cd web

# Install dependencies
npm install

# Generate TypeScript API client from OpenAPI spec
npm run codegen

# Run development server with HMR
npm run dev

# Build production bundle
npm run build

# Run linter
npm run lint
```

### Go Development
```bash
# Build Go binary directly (without Docker)
go build -o bin/ cmd/pixelgw/pixelgw.go

# Generate Go API code from OpenAPI spec
go generate internal/api/pixelgw.go
```

## Architecture

### Core Components

**WebSocket Hub Architecture**
- `internal/hub/hub.go`: Central connection manager coordinating all WebSocket clients and channels
- `internal/hub/client.go`: Individual device WebSocket connection handling
- `internal/hub/channel.go`: Channel lifecycle, applet execution, and image broadcasting
- `internal/hub/task.go`: Asynchronous task processing system for thread-safe operations

**API Layer**
- `pixelgw.yaml`: OpenAPI 3.0 specification defining the REST API contract
- `internal/api/pixelgw.gen.go`: Auto-generated server stubs from OpenAPI spec
- `internal/api/`: Handler implementations for channels, devices, applets, sessions

**Data Persistence**
- SQLite database with Canonical/sqlair ORM
- Schema: channels → channel_applets, devices (with channel subscriptions)
- Automatic schema migrations on startup

**Applet System**
- `internal/catalog/`: Discovers and manages Pixlet apps from `/apps` directories
- Apps sourced from Tidbyt community repository (git submodule) and local directory
- Pixlet runtime (`tidbyt.dev/pixlet`) executes Starlark scripts to generate images

### WebSocket Protocol
Devices connect via: `ws://server:8080/ws?device=<deviceUUID>`
- Automatic device registration on first connection
- Real-time image streaming to subscribed devices
- Channel-based broadcasting to device groups

### Code Generation Pipeline
1. Modify `pixelgw.yaml` (OpenAPI spec)
2. Run `make generate` to update Go server code
3. Run `npm run codegen` in web/ to update TypeScript client
4. Changes automatically integrated into build process

## Important Development Notes

- **Single-tenant design**: No authentication by design (trusted network assumption)
- **Git submodules**: Initialize community apps with `git submodule init && git submodule update`
- **Docker-first workflow**: Use Makefile commands for consistent development environment
- **Port allocation**: Development (8081), Production (8080)
- **Frontend SPA**: React Router with fallback handling in Go server
- **Auto-generated code**: Never manually edit `*.gen.go` or `openapi/index.ts` files

## Current Limitations
- No OAuth parameter support for applets requiring authentication
- No audio support for Tidbyt2 devices
- No automated testing framework
- Limited error handling and logging infrastructure