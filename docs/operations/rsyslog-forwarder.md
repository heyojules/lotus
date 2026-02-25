# Durable Local Forwarding With rsyslog

This guide replaces direct `app | lotus` pipelines for production durability.

Recommended topology:

`app -> journald -> rsyslog (disk queue) -> lotus tcp:4000`

## Why This Topology

- Direct Unix pipes are not durable when Lotus restarts.
- rsyslog action queues can persist to disk and retry until Lotus is back.
- Keeps the stack simple: no broker, no external queue service.

## Prerequisites

- Lotus service is running locally and listening on `127.0.0.1:4000`.
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
sudo install -m 0644 configs/rsyslog/lotus-local-forwarder.conf /etc/rsyslog.d/20-lotus-forwarder.conf
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
systemctl is-active lotus
```

## Smoke Test

Send one test line through syslog:

```sh
logger -t lotus-test "hello from rsyslog"
```

Then verify in Lotus using your normal query flow (`/api/query` or `lotus-tui`).

## Failure Drill (Backlog + Drain)

This validates durability during Lotus downtime.

1. Stop Lotus:

```sh
sudo systemctl stop lotus
```

2. Generate load while Lotus is down:

```sh
for i in $(seq 1 20000); do logger -t lotus-drill "rsyslog durability test $i"; done
```

3. Confirm rsyslog queue files are present:

```sh
sudo ls -lah /var/spool/rsyslog | rg lotus_fwd
```

4. Start Lotus again:

```sh
sudo systemctl start lotus
```

5. Verify backlog drains:

- rsyslog logs no longer show suspended forward action.
- Lotus query counts continue increasing until caught up.

## Operational Guardrails

- Watch spool disk usage:
  - `/var/spool/rsyslog`
- Set `queue.maxDiskSpace` based on host disk budget and peak outage tolerance.
- Keep Lotus bound to localhost unless cross-host ingest is required.
- Expect occasional duplicates on reconnect boundaries; dedup can be added later in Lotus.

## References

- rsyslog queue concepts: <https://www.rsyslog.com/doc/concepts/queues.html>
- rsyslog queue parameters: <https://www.rsyslog.com/doc/rainerscript/queue_parameters.html>
- rsyslog `omfwd`: <https://www.rsyslog.com/doc/modules/omfwd.html>
- Linux pipe behavior: <https://man7.org/linux/man-pages/man7/pipe.7.html>
