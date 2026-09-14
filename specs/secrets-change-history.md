# Specification: Secrets Change History

## 1. Objective
Track edits to stored credential entries in the `secrets` manager so changes to
passwords and other fields are auditable and recoverable. Today `set` only bumps
`entries.updated_at`; the previous value is lost, and `secrets history` covers
only DB-user provisioning. This feature adds a per-entry, per-field change log.

## 2. Scope

### 2.1 In scope
- Record create / update / delete events for entries managed by `shared/secrets`.
- Per-field diffing: capture the old and new value of every field that changed.
- A CLI to view an entry's change history, filterable by field and time.
- Retention controls to bound DB growth.
- Backfill of a synthetic import event for entries that predate the feature.

### 2.2 Out of scope
- Encryption at rest (the DB remains plaintext with `700`/`600` perms).
- Remote/centralized audit shipping.
- Tracking changes made outside the CLI (raw `sqlite3`, `import-postgres-creds.sh`,
  `migrate_legacy`, or the `init-db.sh` bootstrap). These paths bypass the history
  writer; see §5.3.
- Password rotation against external services.

## 3. Data Model

New table created in `init_db()` (idempotent `CREATE TABLE IF NOT EXISTS`):

```sql
CREATE TABLE IF NOT EXISTS entry_history (
    id          INTEGER PRIMARY KEY,
    entry_id    INTEGER NOT NULL,           -- entries.id at time of change
    entry_name  TEXT    NOT NULL,           -- denormalized; survives delete
    field       TEXT,                       -- NULL for snapshot events
    old_value   TEXT,
    new_value   TEXT,
    change_type TEXT NOT NULL,              -- create|update|delete|import|prune
    changed_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_history_entry ON entry_history(entry_id, changed_at);
CREATE INDEX IF NOT EXISTS idx_history_name  ON entry_history(entry_name, changed_at);
```

`entry_name` is denormalized so history remains readable after an entry is
deleted and so it survives a future rename feature.

### 3.1 Event semantics
| change_type | rows written |
|---|---|
| `create` | one row per populated field; `old_value` NULL |
| `update` | one row per field whose value actually changed; `old_value` + `new_value` set |
| `delete` | one row with `field` NULL and `old_value` = JSON snapshot of the entry |
| `import` | one synthetic row (`field` NULL, values NULL); emits `new_value` = JSON of non-secret metadata only |
| `prune` | written when retention removes rows (see §5.3) |

No row is written for a no-op `set` where every field equals its current value.

## 4. Writer

Add a single helper used by every mutating command:

```python
def record_change(conn, entry, change_type, diffs=None, snapshot=None):
    """Append history rows. `entry` is the row before/after mutation as appropriate."""
```

- `cmd_upsert` (both `add` and `set`) computes the diff against the existing row
  and calls `record_change`, inside the same transaction as the entry write.
- `cmd_rm` calls `record_change(..., "delete", snapshot=json.dumps(dict(row)))`.
- Writes are committed atomically with the entry mutation so history cannot
  diverge from the live table.

## 5. CLI

### 5.1 Viewing
New command `secrets changes`:

```
secrets changes [<name>] [--field F] [--limit N] [--all] [--json] [--reveal]
```

- No `<name>` -> all entries, newest first.
- `<name>` -> history for that entry, resolved through the existing `resolve()`
  legacy suffix handling.
- `--field F` -> only rows for that field.
- `--limit N` -> max rows (default 50).
- `--all` -> no implicit limit.
- `--json` -> machine-readable output including `entry_name`, `field`,
  `old_value`, `new_value`, `change_type`, `changed_at`.
- `--reveal` -> show values; without it old/new values are masked.

Backward compatibility: the existing `secrets history` command (DB-user creation
audit) is unchanged. Keeping the two commands separate avoids breaking scripts
that parse `history` output. Cross-reference each command from the other's help.

### 5.2 Retention
```
secrets changes prune [--keep N] [--older-than DAYS] [--yes]
```
- Keeps the newest `N` rows per entry (default from `SECRETS_HISTORY_KEEP`, else 100).
- `--older-than` prunes by age; combinable with `--keep` (age applies first).
- Prompts unless `--yes`; writes a `prune` marker row.
- If both options are omitted, prune uses the default `--keep` value.

### 5.3 Opt-out / bypass
- Environment `SECRETS_HISTORY=0` disables history writes for that invocation.
- Direct SQL paths (`migrate_legacy`, `import-postgres-creds.sh`) do not call the
  writer and therefore produce no history. This is accepted and documented.

### 5.4 Output
Default human view, grouped by timestamp:

```
2026-09-14 23:27:54  update  password   ****  ->  ****
2026-09-14 22:33:05  create  username   manaseslovera07@gmail.com
2026-09-14 22:33:05  create  password   ****
```

Values are masked as `****` unless `--reveal`. Passwords are masked even in
`--json` unless `--reveal` is passed.

## 6. Security
- History stores old and new values in plaintext, consistent with the existing
  DB threat model (owner-only filesystem permissions). This enables password
  recovery, which is the primary motivation.
- `--reveal` is required to display values in any output, including `--json`.
- `secrets changes` is never included in `export` output; exporting history is
  out of scope for this spec.
- Permissions are re-enforced (`enforce_perms()`) after any command that writes
  history.

## 7. Migration
On first run after upgrade, `init_db()` creates `entry_history` and backfills one
`import` row per existing entry when the table was just created and is empty:

- `changed_at = entries.created_at`
- `change_type = 'import'`
- no field values (avoids inventing unknown history)
- `new_value` = JSON of `{type, url, description}` only

Backfill runs once; it is skipped if `entry_history` already has rows.

## 8. Edge Cases
- `set` with only `--username`: records a single `username` diff.
- `set` on a non-existent name creates the entry (`create` events).
- `set` with an unchanged generated password must not create spurious rows.
- Field set to empty string is a real change and is recorded.
- Delete followed by re-add of the same name produces history under the same
  `entry_name` but a new `entry_id`; both are shown when querying by name.
- `resolve()` legacy names (`*_password`, `*_user`) map to the underlying entry id.

## 9. Acceptance Criteria
1. Updating a password via `secrets set` produces exactly one `update` history row
   for `field='password'` with the correct old/new values.
2. `secrets changes <name>` shows the change without exposing values; adding
   `--reveal` shows them.
3. `secrets add` emits `create` rows for each populated field.
4. `secrets rm` emits a single `delete` snapshot row, and the snapshot survives
   the entry's deletion.
5. No-op `set` writes no history rows.
6. `secrets changes prune --keep N --yes` bounds each entry to N rows and records
   a `prune` marker.
7. Existing `secrets history`, `list`, `get`, `show`, `export` behavior is
   unchanged and all existing tests/scripts pass.
8. DB directory/file permissions remain `700`/`600` after history writes.
9. `SECRETS_HISTORY=0` suppresses history writes.

## 10. Documentation
- Update `secrets help` and README §"The `secrets` CLI" with `changes` and
  `changes prune`.
- Note the plaintext-history tradeoff and the opt-out env var.
