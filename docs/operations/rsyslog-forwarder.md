# Durable Local Forwarding With rsyslog

This guide replaces direct `app | tiny-telemetry` pipelines for production durability.

Recommended topology:

`app -> journald -> rsyslog (disk queue) -> tiny-telemetry tcp:4000`

## Why This Topology

- Direct Unix pipes are not durable when Tiny Telemetry restarts.
- rsyslog action queues can persist to disk and retry until Tiny Telemetry is back.
- Keeps the stack simple: no broker, no external queue service.

## Prerequisites

- Tiny Telemetry service is running locally and listening on `127.0.0.1:4000`.
- rsyslog installed and enabled.
- Persistent journald storage enabled (recommended).

Optional journald persistence (`/etc/systemd/journald.conf`):

```ini
Storage=persistent
SystemMaxUse=2G
```

After changes:

```sh
sudo systemctl restart systemd-journald
```

## Install Forwarder Config

1. Copy the template:

```sh
sudo install -m 0644 configs/rsyslog/tiny-telemetry-local-forwarder.conf /etc/rsyslog.d/20-tiny-telemetry-forwarder.conf
```

2. Validate config syntax:

```sh
sudo rsyslogd -N1
```

3. Restart rsyslog:

```sh
sudo systemctl restart rsyslog
```

4. Confirm services:

```sh
systemctl is-active rsyslog
systemctl is-active tiny-telemetry
```

## Smoke Test

Send one test line through syslog:

```sh
logger -t tiny-telemetry-test "hello from rsyslog"
```

Then verify in Tiny Telemetry using your normal query flow (`/api/query` or `tiny-telemetry-tui`).

## Failure Drill (Backlog + Drain)

This validates durability during Tiny Telemetry downtime.

1. Stop Tiny Telemetry:

```sh
sudo systemctl stop tiny-telemetry
```

2. Generate load while Tiny Telemetry is down:

```sh
for i in $(seq 1 20000); do logger -t tiny-telemetry-drill "rsyslog durability test $i"; done
```

3. Confirm rsyslog queue files are present:

```sh
sudo ls -lah /var/spool/rsyslog | rg tiny_telemetry_fwd
```

4. Start Tiny Telemetry again:

```sh
sudo systemctl start tiny-telemetry
```

5. Verify backlog drains:

- rsyslog logs no longer show suspended forward action.
- Tiny Telemetry query counts continue increasing until caught up.

## Operational Guardrails

- Watch spool disk usage:
  - `/var/spool/rsyslog`
- Set `queue.maxDiskSpace` based on host disk budget and peak outage tolerance.
- Keep Tiny Telemetry bound to localhost unless cross-host ingest is required.
- Expect occasional duplicates on reconnect boundaries; dedup can be added later in Tiny Telemetry.

## References

- rsyslog queue concepts: <https://www.rsyslog.com/doc/concepts/queues.html>
- rsyslog queue parameters: <https://www.rsyslog.com/doc/rainerscript/queue_parameters.html>
- rsyslog `omfwd`: <https://www.rsyslog.com/doc/modules/omfwd.html>
- Linux pipe behavior: <https://man7.org/linux/man-pages/man7/pipe.7.html>
