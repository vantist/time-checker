## 1. Export GetAntigravityModel and Add Tests

- [x] 1.1 In `internal/transcript/antigravity_transcript_test.go`, add unit test cases to verify the exported `GetAntigravityModel` behavior (e.g. parsing model when settings.json exists).
- [x] 1.2 Export `getAntigravityModel` to `GetAntigravityModel` in `internal/transcript/antigravity_transcript.go`. Update existing references in `internal/transcript/antigravity_transcript.go` and `internal/transcript/antigravity_transcript_test.go`. Ensure tests pass.

## 2. Project Path Fallback and Default Model in resolvePromptInput

- [ ] 2.1 In `cmd/tt/record_test.go`, write test cases for `resolvePromptInput` verifying:
  - If project path is not provided via command-line flags and stdin payload is missing, it falls back to the current directory (`os.Getwd()`).
  - If the integration tool is `"antigravity"` and model is empty, it attempts to load the default model from configurations.
- [ ] 2.2 In `cmd/tt/record.go` (`resolvePromptInput`), implement the project path fallback to `os.Getwd()`.
- [ ] 2.3 In `cmd/tt/record.go` (`resolvePromptInput`), implement loading and resolving default model for `"antigravity"` using `transcript.GetAntigravityModel(nil)` when the model is empty. Ensure all tests in `cmd/tt` pass.

## 3. Reconcile Sessions Model Backfill

- [ ] 3.1 In `internal/reconcile/reconcile_test.go`, add a failing test verifying that when a turn's model is reconciled, the corresponding session's model is updated/backfilled if it was previously empty or NULL.
- [ ] 3.2 In `internal/reconcile/reconcile.go` (`reconcileTurn`), implement backfilling the `sessions.model` field within the database transaction when a non-empty turn model is found and the session's model is empty or NULL. Ensure all tests in `internal/reconcile` pass.
