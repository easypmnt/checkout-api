#!/bin/bash
go mod download
go build -o ./bin/migrate ./cmd/migrate/
go build -o ./bin/api ./cmd/api/
