package duckdb

import "github.com/control-theory/lotus/internal/model"

// Type aliases re-export model types so existing duckdb.Store method
// signatures remain valid without changing every call site at once.
type LogRecord = model.LogRecord
type WordCount = model.WordCount
type AttributeStat = model.AttributeStat
type AttributeKeyStat = model.AttributeKeyStat
type DimensionCount = model.DimensionCount
type MinuteCounts = model.MinuteCounts
