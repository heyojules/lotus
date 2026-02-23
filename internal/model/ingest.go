package model

// IngestEnvelope carries one raw log line with source metadata.
// It is the transport contract between ingestion plugins and processing.
type IngestEnvelope struct {
	Source string
	Line   string
}
