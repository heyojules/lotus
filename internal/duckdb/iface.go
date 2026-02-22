package duckdb

import "github.com/control-theory/lotus/internal/model"

// Type aliases re-export model interfaces and types so existing
// consumers that import duckdb for these continue to compile.
type QueryOpts = model.QueryOpts
type LogQuerier = model.LogQuerier
type SchemaQuerier = model.SchemaQuerier
