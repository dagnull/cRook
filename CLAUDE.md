# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**cRook** is a Go project. The repository is in early development — no code has been committed yet.

## Commands

```bash
# Initialize module (if not yet done)
go mod init github.com/dagnull/cRook

# Build
go build ./...

# Test
go test ./...

# Run a single test
go test ./path/to/package -run TestName

# Lint (requires golangci-lint)
golangci-lint run

# Format
go fmt ./...
goimports -w .
```