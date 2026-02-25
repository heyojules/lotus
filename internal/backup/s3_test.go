package backup

import (
	"strings"
	"testing"
)

func TestParseS3BucketURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		raw       string
		wantErr   bool
		wantBkt   string
		wantPre   string
		errSubstr string
	}{
		{
			name:    "bucket only",
			raw:     "s3://my-bucket",
			wantBkt: "my-bucket",
			wantPre: "",
		},
		{
			name:    "bucket with prefix",
			raw:     "s3://my-bucket/lotus/backups",
			wantBkt: "my-bucket",
			wantPre: "lotus/backups",
		},
		{
			name:      "invalid scheme",
			raw:       "https://my-bucket/lotus",
			wantErr:   true,
			errSubstr: "s3:// scheme",
		},
		{
			name:      "missing bucket",
			raw:       "s3:///lotus",
			wantErr:   true,
			errSubstr: "missing bucket",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotBkt, gotPre, err := parseS3BucketURL(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("err = %q, want substring %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseS3BucketURL error: %v", err)
			}
			if gotBkt != tt.wantBkt {
				t.Fatalf("bucket = %q, want %q", gotBkt, tt.wantBkt)
			}
			if gotPre != tt.wantPre {
				t.Fatalf("prefix = %q, want %q", gotPre, tt.wantPre)
			}
		})
	}
}

func TestNewS3Uploader_MissingCredentials(t *testing.T) {
	t.Parallel()

	_, err := NewS3Uploader(S3Config{
		BucketURL: "s3://my-bucket/lotus",
		Endpoint:  "s3.amazonaws.com",
		UseSSL:    true,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
