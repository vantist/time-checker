package transcript

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// ParseAntigravityLog parses the transcript.jsonl file and returns main agent model usage.
func ParseAntigravityLog(path string) (WindowResult, error) {
	return ParseAntigravityLogWindow(path, 0, -1)
}

// ParseAntigravityLogWindow parses the transcript.jsonl file and returns main agent model usage
// restricting the SQLite metrics to the steps contained within the transcript line range [fromOffset, toOffset).
// If toOffset is -1, it reads to the end of the file.
func ParseAntigravityLogWindow(path string, fromOffset, toOffset int) (WindowResult, error) {
	all, err := loadTranscript(path)
	if err != nil {
		return WindowResult{}, err
	}

	mainModel := GetAntigravityModel(all)

	end := len(all)
	if toOffset != -1 && toOffset < end {
		end = toOffset
	}
	if fromOffset > end {
		fromOffset = end
	}

	var stepIndices []int
	for i := fromOffset; i < end; i++ {
		if all[i].StepIndex > 0 {
			stepIndices = append(stepIndices, all[i].StepIndex)
		}
	}

	// Retrieve session ID from path to query SQLite database
	sessionID := getSessionID(path)
	var input, output, cacheRead int
	var cacheCreation, cache5m, cache1h int
	if sessionID != "" {
		if len(stepIndices) > 0 || (fromOffset == 0 && toOffset == -1) {
			input, output, cacheRead, _ = queryAntigravityTokens(sessionID, stepIndices)
		}
	}

	// Fallback to JSONL sum if SQLite query returned 0 tokens
	if input == 0 && output == 0 {
		acc := sumWindow(all, fromOffset, end)
		input = acc.InputTokens
		output = acc.OutputTokens
		cacheRead = acc.CacheReadInputTokens
		cacheCreation = acc.CacheCreationInputTokens
		cache5m = acc.CacheCreation.Ephemeral5m
		cache1h = acc.CacheCreation.Ephemeral1h
	}

	var result WindowResult
	result.Usages = append(result.Usages, ModelUsage{
		Model:               mainModel,
		IsSubagent:          false,
		InputTokens:         input,
		OutputTokens:        output,
		CacheReadTokens:     cacheRead,
		CacheCreationTokens: cacheCreation,
		CacheCreation5m:     cache5m,
		CacheCreation1h:     cache1h,
	})

	// Also extract subagents if any (noting that Antigravity subagent capture uses standard transcript patterns if present)
	subUsages := extractSubagentModelUsages(path, all, fromOffset, end)
	result.Usages = append(result.Usages, subUsages...)

	return result, nil
}

// GetAntigravityModel reads and resolves the model name from ~/.gemini/antigravity-cli/settings.json,
// falling back to ~/.gemini/antigravity/settings.json, and normalizes it.
func GetAntigravityModel(all []entry) string {
	if logModel := findMainModel(all); logModel != "unknown" {
		return logModel
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "gemini-3.5-flash"
	}

	paths := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "settings.json"),
		filepath.Join(home, ".gemini", "antigravity", "settings.json"),
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(data, &cfg); err == nil && cfg.Model != "" {
			return cleanAntigravityModel(cfg.Model)
		}
	}

	return "gemini-3.5-flash"
}

func cleanAntigravityModel(name string) string {
	name = strings.ToLower(name)
	// Strip parentheses e.g. (medium) or (large)
	if i := strings.Index(name, "("); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimSpace(name)
	// Replace spaces/hyphens with a single hyphen, keeping alphanumeric characters and dots
	var result []rune
	lastIsDash := false
	for _, r := range name {
		if r == ' ' || r == '-' {
			if !lastIsDash {
				result = append(result, '-')
				lastIsDash = true
			}
		} else if r == '.' {
			result = append(result, r)
			lastIsDash = false
		} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result = append(result, r)
			lastIsDash = false
		}
	}
	name = string(result)
	name = strings.Trim(name, "-")
	if name == "" {
		return "gemini-3.5-flash"
	}
	return name
}

func getSessionID(path string) string {
	abs, err := filepath.Abs(expandHome(path))
	if err != nil {
		abs = path
	}
	parts := strings.Split(abs, string(filepath.Separator))
	for i := len(parts) - 1; i >= 3; i-- {
		if parts[i] == "logs" && parts[i-1] == ".system_generated" && parts[i-3] == "brain" {
			return parts[i-2]
		}
	}
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "brain" {
			return parts[i+1]
		}
	}
	return ""
}

func queryAntigravityTokens(sessionID string, stepIndices []int) (int, int, int, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, 0, 0, err
	}
	dbPaths := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "conversations", sessionID+".db"),
		filepath.Join(home, ".gemini", "antigravity", "conversations", sessionID+".db"),
	}

	var dbPath string
	for _, p := range dbPaths {
		if _, err := os.Stat(p); err == nil {
			dbPath = p
			break
		}
	}
	if dbPath == "" {
		return 0, 0, 0, fmt.Errorf("database not found for session %s", sessionID)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0, 0, 0, err
	}
	defer db.Close()

	var rows *sql.Rows
	if len(stepIndices) == 0 {
		rows, err = db.Query("SELECT data FROM gen_metadata")
	} else {
		placeholders := make([]string, len(stepIndices))
		args := make([]interface{}, len(stepIndices))
		for i, idx := range stepIndices {
			placeholders[i] = "?"
			args[i] = idx
		}
		query := fmt.Sprintf("SELECT data FROM gen_metadata WHERE idx IN (%s)", strings.Join(placeholders, ","))
		rows, err = db.Query(query, args...)
	}
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	var totalInput, totalOutput, totalCacheRead int
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			continue
		}
		inp, out, cache := parseUsageFromProtobuf(data)
		totalInput += inp
		totalOutput += out
		totalCacheRead += cache
	}
	return totalInput, totalOutput, totalCacheRead, nil
}

func parseUsageFromProtobuf(data []byte) (input, output, cacheRead int) {
	var parse func(buf []byte, path []int)
	parse = func(buf []byte, path []int) {
		i := 0
		for i < len(buf) {
			tag, n := readVarint(buf[i:])
			if n <= 0 {
				break
			}
			i += n

			fieldNum := int(tag >> 3)
			wireType := int(tag & 0x07)

			switch wireType {
			case 0: // Varint
				val, n := readVarint(buf[i:])
				if n <= 0 {
					return
				}
				i += n

				if len(path) == 2 && path[0] == 1 && path[1] == 4 {
					switch fieldNum {
					case 2:
						input = int(val)
					case 3:
						output = int(val)
					case 5:
						cacheRead = int(val)
					}
				} else if len(path) == 3 && path[0] == 1 && path[1] == 17 && path[2] == 2 {
					switch fieldNum {
					case 2:
						input = int(val)
					case 3:
						output = int(val)
					case 5:
						cacheRead = int(val)
					}
				}
			case 1: // 64-bit
				i += 8
			case 2: // Length-delimited
				length, n := readVarint(buf[i:])
				if n <= 0 {
					return
				}
				i += n
				if i+int(length) > len(buf) {
					return
				}
				subBuf := buf[i : i+int(length)]
				i += int(length)

				parse(subBuf, append(path, fieldNum))
			case 5: // 32-bit
				i += 4
			default:
				return
			}
		}
	}

	parse(data, []int{})
	return
}

func readVarint(buf []byte) (uint64, int) {
	var val uint64
	var shift uint
	for i, b := range buf {
		val |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return val, i + 1
		}
		shift += 7
		if shift >= 64 {
			break
		}
	}
	return 0, 0
}
