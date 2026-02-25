package backup

import (
	"context"
	"time"
)

// Config controls periodic DuckDB backups.
type Config struct {
	Enabled   bool
	Interval  time.Duration
	LocalDir  string
	KeepLast  int
	BucketURL string

	S3Endpoint     string
	S3Region       string
	S3AccessKey    string
	S3SecretKey    string
	S3SessionToken string
	S3UseSSL       bool
}

// Snapshotter is the minimal DB snapshot contract used by BackupManager.
type Snapshotter interface {
	DBPath() string
	SnapshotTo(dstPath string) error
}

// Uploader uploads one backup artifact.
type Uploader interface {
	UploadFile(ctx context.Context, localPath string) error
}
