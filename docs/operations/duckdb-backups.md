# DuckDB Backup Strategy (Initial POC)

This is the first production-friendly backup path for Lotus.

Default behavior:

- Backups are disabled.
- No extra IO or remote calls are performed.

When enabled:

1. Lotus creates periodic local snapshots of the DuckDB file.
2. Snapshots are rotated locally (`backup-keep-last`).
3. If bucket config is present, each snapshot is uploaded to S3-compatible storage.
   Current POC upload path uses `aws s3 cp` (AWS CLI) from the Lotus process.

## Configuration

```yaml
backup-enabled: true
backup-interval: 6h
backup-local-dir: ~/.local/share/lotus/backups
backup-keep-last: 24

# Optional remote upload
backup-bucket-url: s3://my-bucket/lotus-backups
backup-s3-endpoint: s3.amazonaws.com
backup-s3-region: us-east-1
backup-s3-access-key: your-access-key
backup-s3-secret-key: your-secret-key
backup-s3-session-token: "" # optional
backup-s3-use-ssl: true
```

## POC Design Choices

- Simplicity first:
  - single process, in-app scheduler
  - no sidecar or external backup daemon
- Safety:
  - snapshot operation performs `CHECKPOINT` before copying DB file
  - local retention keeps bounded disk usage
- Compatibility:
  - remote path uses S3-compatible API with static credentials
  - bucket URL uses `s3://bucket/prefix`
  - upload command uses local AWS CLI binary (no extra Go SDK dependency)

## Recommended Initial Policy

- `backup-interval`: `6h`
- `backup-keep-last`: `24` (about 6 days of local history)
- Run one restore drill per environment before relying on backups.

## Operational Checks

- Ensure `backup-local-dir` has free space and permissions.
- Watch Lotus logs for `backup:` lines (snapshot/upload/prune failures).
- Confirm objects arrive in bucket prefix.
- Ensure `aws` CLI is available in `PATH` when remote upload is enabled.

## Caveats (Current POC)

- Upload is best-effort per run; failed uploads are not queued/retried out of band.
- Encryption-at-rest/KMS lifecycle rules are delegated to bucket policy.
- Restore tooling is manual for now (copy snapshot file back as DB path).

## Next Iterations

1. Add restore command (`lotus backup restore --from ...`).
2. Add upload retry queue and exponential backoff.
3. Add checksum manifest file per snapshot.
4. Add optional object lock/immutability guidance.
