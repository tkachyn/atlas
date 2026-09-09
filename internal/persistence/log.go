package persistence

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tkachyn/atlas/internal/command"
	"github.com/tkachyn/atlas/internal/protocol"
	"github.com/tkachyn/atlas/internal/store"
)

const recordVersion = "A1"

// log stores commands in an append-only file
type Log struct {
	mu   sync.Mutex
	path string
	file *os.File
}

// open creates or opens an append-only command log
func Open(path string) (*Log, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create persistence directory: %w", err)
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open persistence log: %w", err)
	}

	return &Log{path: path, file: file}, nil
}

// append writes a checksummed command to the log and flushes it to disk
func (l *Log) Append(cmd protocol.Command) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.appendLocked(cmd); err != nil {
		return err
	}
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("sync command log: %w", err)
	}

	return nil
}

// replay applies every logged command to the store
func (l *Log) Replay(data *store.Store) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek persistence log: %w", err)
	}

	contents, err := io.ReadAll(l.file)
	if err != nil {
		return fmt.Errorf("read persistence log: %w", err)
	}

	offset := 0
	for offset < len(contents) {
		relativeEnd := bytes.IndexByte(contents[offset:], '\n')
		if relativeEnd < 0 {
			// discard only the incomplete final record after an interrupted write
			if err := l.truncateLocked(int64(offset)); err != nil {
				return fmt.Errorf("truncate partial record: %w", err)
			}
			break
		}

		end := offset + relativeEnd + 1
		line := string(contents[offset:end])
		cmd, err := decodeRecord(line)
		if err != nil {
			return fmt.Errorf("replay record at byte %d: %w", offset, err)
		}

		command.Execute(cmd, data)
		offset = end
	}

	if _, err := l.file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("restore persistence position: %w", err)
	}

	return nil
}

// compact replaces the log with the current live store contents
func (l *Log) Compact(data *store.Store) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.compactLocked(data)
}

// compact the log when it reaches the configured size
func (l *Log) MaybeCompact(maxBytes int64, data *store.Store) error {
	if maxBytes <= 0 {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	info, err := l.file.Stat()
	if err != nil {
		return fmt.Errorf("stat persistence log: %w", err)
	}
	if info.Size() < maxBytes {
		return nil
	}

	return l.compactLocked(data)
}

// close closes the persistence log
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close persistence log: %w", err)
	}

	return nil
}

func (l *Log) appendLocked(cmd protocol.Command) error {
	if cmd.Name == "EXPIRE" {
		// store an absolute deadline so replay does not reset the key lifetime
		seconds, err := strconv.ParseInt(cmd.Args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("parse expiration: %w", err)
		}
		cmd = protocol.Command{
			Name: "EXPIREAT",
			Args: []string{
				cmd.Args[0],
				strconv.FormatInt(time.Now().Add(time.Duration(seconds)*time.Second).UnixNano(), 10),
			},
		}
	}

	if _, err := l.file.WriteString(encodeRecord(cmd)); err != nil {
		return fmt.Errorf("append command: %w", err)
	}

	return nil
}

func (l *Log) compactLocked(data *store.Store) error {
	tempPath := l.path + ".tmp"
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create compacted log: %w", err)
	}

	snapshot := data.Snapshot()
	// sort entries so equivalent compactions produce stable files
	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].Key < snapshot[j].Key
	})

	for _, item := range snapshot {
		if _, err := tempFile.WriteString(encodeRecord(protocol.Command{
			Name: "SET",
			Args: []string{item.Key, item.Value},
		})); err != nil {
			tempFile.Close()
			return fmt.Errorf("write compacted value: %w", err)
		}
		if !item.ExpiresAt.IsZero() {
			if _, err := tempFile.WriteString(encodeRecord(protocol.Command{
				Name: "EXPIREAT",
				Args: []string{item.Key, strconv.FormatInt(item.ExpiresAt.UnixNano(), 10)},
			})); err != nil {
				tempFile.Close()
				return fmt.Errorf("write compacted expiration: %w", err)
			}
		}
	}

	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("sync compacted log: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close compacted log: %w", err)
	}
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close old persistence log: %w", err)
	}
	if err := os.Remove(l.path); err != nil {
		return fmt.Errorf("remove old persistence log: %w", err)
	}
	if err := os.Rename(tempPath, l.path); err != nil {
		return fmt.Errorf("replace persistence log: %w", err)
	}

	l.file, err = os.OpenFile(l.path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("reopen compacted log: %w", err)
	}

	return nil
}

func (l *Log) truncateLocked(size int64) error {
	if err := l.file.Truncate(size); err != nil {
		return err
	}
	if err := l.file.Sync(); err != nil {
		return err
	}
	return nil
}

func encodeRecord(cmd protocol.Command) string {
	payload := protocol.Format(cmd)
	checksum := crc32.ChecksumIEEE([]byte(payload))
	return fmt.Sprintf("%s|%08x|%s\n", recordVersion, checksum, payload)
}

func decodeRecord(line string) (protocol.Command, error) {
	payload := strings.TrimSuffix(line, "\n")
	if !strings.HasPrefix(payload, recordVersion+"|") {
		// accept legacy plain-text records during recovery
		return protocol.Parse(payload)
	}

	parts := strings.SplitN(payload, "|", 3)
	if len(parts) != 3 {
		return protocol.Command{}, fmt.Errorf("malformed record")
	}

	expected, err := strconv.ParseUint(parts[1], 16, 32)
	if err != nil {
		return protocol.Command{}, fmt.Errorf("invalid checksum")
	}
	actual := crc32.ChecksumIEEE([]byte(parts[2]))
	if uint32(expected) != actual {
		return protocol.Command{}, fmt.Errorf("checksum mismatch")
	}

	return protocol.Parse(parts[2])
}
