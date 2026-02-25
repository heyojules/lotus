package backup

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
)

// S3Config holds S3 uploader parameters for backup uploads.
type S3Config struct {
	BucketURL    string
	Endpoint     string
	Region       string
	AccessKey    string
	SecretKey    string
	SessionToken string
	UseSSL       bool
	ContentType  string // reserved for future native uploader
}

// S3Uploader uploads backup files using AWS CLI (`aws s3 cp`).
// This keeps the initial POC dependency-free in Go code.
type S3Uploader struct {
	bucket    string
	keyPrefix string
	cfg       S3Config
}

// NewS3Uploader constructs an uploader from an S3 bucket URL and static credentials.
// BucketURL format: s3://bucket/prefix (prefix optional).
func NewS3Uploader(cfg S3Config) (*S3Uploader, error) {
	bucket, prefix, err := parseS3BucketURL(cfg.BucketURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.AccessKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("s3: access key and secret key are required")
	}
	if _, err := exec.LookPath("aws"); err != nil {
		return nil, fmt.Errorf("s3: aws cli not found in PATH")
	}
	if strings.TrimSpace(cfg.Region) == "" {
		cfg.Region = "us-east-1"
	}
	return &S3Uploader{
		bucket:    bucket,
		keyPrefix: prefix,
		cfg:       cfg,
	}, nil
}

// UploadFile uploads localPath to configured bucket and key prefix.
func (u *S3Uploader) UploadFile(ctx context.Context, localPath string) error {
	objectKey := path.Base(localPath)
	if u.keyPrefix != "" {
		objectKey = path.Join(u.keyPrefix, objectKey)
	}
	dest := fmt.Sprintf("s3://%s/%s", u.bucket, objectKey)

	args := []string{"s3", "cp", localPath, dest, "--region", u.cfg.Region, "--only-show-errors"}
	if endpoint := normalizeEndpoint(u.cfg.Endpoint, u.cfg.UseSSL); endpoint != "" {
		args = append(args, "--endpoint-url", endpoint)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Env = append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+u.cfg.AccessKey,
		"AWS_SECRET_ACCESS_KEY="+u.cfg.SecretKey,
		"AWS_DEFAULT_REGION="+u.cfg.Region,
	)
	if strings.TrimSpace(u.cfg.SessionToken) != "" {
		cmd.Env = append(cmd.Env, "AWS_SESSION_TOKEN="+u.cfg.SessionToken)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("s3 upload command failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func normalizeEndpoint(endpoint string, useSSL bool) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	scheme := "https://"
	if !useSSL {
		scheme = "http://"
	}
	return scheme + endpoint
}

func parseS3BucketURL(raw string) (bucket string, prefix string, err error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", fmt.Errorf("s3: parse bucket-url: %w", err)
	}
	if u.Scheme != "s3" {
		return "", "", fmt.Errorf("s3: bucket-url must use s3:// scheme")
	}
	if strings.TrimSpace(u.Host) == "" {
		return "", "", fmt.Errorf("s3: bucket-url missing bucket name")
	}

	prefix = strings.Trim(strings.TrimSpace(u.Path), "/")
	return u.Host, prefix, nil
}
