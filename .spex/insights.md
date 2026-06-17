# Spex Insights

## [spex-apply] session-identity — 2026-06-18

### Promote candidates

- [ ] macOS `ps` uses `etime=` (HH:MM:SS format), not `etimes=` (seconds, Linux only)
  > **Why**: `ps -p $PID -o etimes=` fails on macOS with "keyword not found". Needed awk parsing to convert HH:MM:SS to seconds. Discovery cost ~20 min during task 5.2.
  > **How to apply**: Any shell script needing process elapsed seconds on macOS: use `ps -p $PID -o etime= | tr -d ' ' | awk -F'[:-]' '{n=NF;s=0;if(n>=1)s+=$n;if(n>=2)s+=$(n-1)*60;if(n>=3)s+=$(n-2)*3600;if(n>=4)s+=$(n-3)*86400;print s}'`

- [ ] Get-or-create DB pattern: return `(id string, err error)` from upsert functions
  > **Why**: UpsertSession needed to return the canonical sessions.id to avoid a second SELECT. Returning the ID from the upsert is cleaner than a follow-up query.
  > **How to apply**: When a DB upsert needs the canonical PK of the affected row, include it in the return signature: `func Upsert(db, row) (id string, err error)`.

- [ ] SQLite `ON CONFLICT` for non-PK UNIQUE constraints requires explicit `UNIQUE INDEX` — without it, SELECT+UPDATE is the correct two-step pattern
  > **Why**: Tried to use INSERT OR IGNORE approach for `(process_pid, process_start)` but the columns lack a UNIQUE constraint (adding one would change schema). SELECT+INSERT-or-UPDATE is cleaner here.
  > **How to apply**: When upsert key is not the PK and adding a UNIQUE constraint is undesirable, use explicit SELECT → branch → INSERT or UPDATE.

### Plan deviations

- Task 6.2 (update SQL grouping) was listed as conditional work but turned out to be N/A: report SQL already uses `sessions.id` as group key, and turns now correctly reference stable ID, so no SQL change was needed.
