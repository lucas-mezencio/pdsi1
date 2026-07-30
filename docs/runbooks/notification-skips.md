# Notification skip runbook

The notification consumer (`internal/infrastructure/scheduler/worker.go`)
records a row in `notification_events` every time a scheduled dose does
**not** actually reach the elderly user. There are two skip statuses:

| Status                       | Meaning                                                                                          |
| ---------------------------- | ------------------------------------------------------------------------------------------------ |
| `skipped_no_tokens`          | The elderly user has zero active device tokens (mobile app never registered, or all disabled).    |
| `skipped_retries_exhausted`  | `sender.Send` returned an error 3 times in a row (FCM outage, Firebase quota, etc). Caregivers may still have received the dose. |

Both rows have `sent_at = now()` and `status != ''` so they are easy to
filter.

## Find which users are skipping doses

```sql
SELECT user_id, status, count(*)
FROM notification_events
WHERE status IN ('skipped_no_tokens', 'skipped_retries_exhausted')
  AND sent_at > now() - interval '24 hours'
GROUP BY user_id, status
ORDER BY count DESC;
```

## Find tokens-less users (likely mobile app registration gap)

```sql
SELECT ne.user_id, count(*) AS missed_doses_last_24h
FROM notification_events ne
WHERE ne.status = 'skipped_no_tokens'
  AND ne.sent_at > now() - interval '24 hours'
GROUP BY ne.user_id
ORDER BY missed_doses_last_24h DESC;
```

Then check the device-token table for that user:

```sql
SELECT id, token, enabled, last_used_at, created_at
FROM user_device_tokens
WHERE user_id = '<uuid>';
```

If the table is empty, the mobile app has never called
`POST /api/v1/users/me/device-tokens` for that user. If rows exist but
`enabled = false`, the user toggled delivery off. If rows exist but
`last_used_at IS NULL`, the token was registered but never used — likely
the FCM token rotated on the client and the new value was never
uploaded.

## Find Firebase-side outages (retries-exhausted spike)

```sql
SELECT date_trunc('minute', sent_at) AS minute, count(*)
FROM notification_events
WHERE status = 'skipped_retries_exhausted'
  AND sent_at > now() - interval '1 hour'
GROUP BY minute
ORDER BY minute DESC;
```

A flat baseline means steady-state FCM errors. A spike points to a
Firebase outage or quota exhaustion.

## Detect the original "no tokens" infinite loop

Before the hotfix in this branch, the consumer Nacked on `no_tokens`
and the watermill redis-stream subscriber re-delivered the same message
in a tight loop (~1ms cadence). Symptom in logs:

```
level=INFO msg="notification no tokens for user <uuid>"   -- many per second
```

After the fix the message is Acked on first encounter, the WARN log fires
exactly once per dose, and a `skipped_no_tokens` row appears in
`notification_events`. A regression to the loop shows up as `INFO`
re-appearing at high frequency — alert on `rate(log{msg="notification no
tokens for user"}[5m]) > 10`.