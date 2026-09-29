package logging

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TailFiles returns up to the last `lines` lines from the .log files under dir,
// oldest file first. A missing directory yields an empty result, not an error.
func TailFiles(dir string, lines int) ([]string, error) {
	if lines <= 0 {
		lines = 100
	}
	files, err := logFiles(dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return []string{}, nil
	}

	collected := make([]string, 0, lines)
	for i := len(files) - 1; i >= 0 && len(collected) < lines; i-- {
		data, err := os.ReadFile(files[i])
		if err != nil {
			continue
		}
		fileLines := splitLines(data)
		need := lines - len(collected)
		if need > len(fileLines) {
			need = len(fileLines)
		}
		part := fileLines[len(fileLines)-need:]
		collected = append(part, collected...)
	}
	return collected, nil
}

// Tailer emits the complete lines appended to the newest log file under dir.
// It is used to implement `logs --follow`.
type Tailer struct {
	dir    string
	file   string
	offset int64
}

// NewTailer creates a tailer for the log directory of a single process.
func NewTailer(dir string) *Tailer {
	return &Tailer{dir: dir}
}

// SeekEnd positions the tailer at the end of the newest existing log file so
// only lines written afterwards are returned by Next.
func (t *Tailer) SeekEnd() error {
	files, err := logFiles(t.dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	newest := files[len(files)-1]
	info, err := os.Stat(newest)
	if err != nil {
		return err
	}
	t.file = newest
	t.offset = info.Size()
	return nil
}

// Next returns the complete lines appended since the previous call.
func (t *Tailer) Next() ([]string, error) {
	files, err := logFiles(t.dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}
	newest := files[len(files)-1]
	// A rotation created a new file: restart from its beginning so no line is lost.
	if newest != t.file {
		t.file = newest
		t.offset = 0
	}

	f, err := os.Open(newest)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Seek(t.offset, io.SeekStart); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// Only consume up to the last newline; keep any partial line for next time.
	idx := bytes.LastIndexByte(data, '\n')
	if idx < 0 {
		return nil, nil
	}
	complete := data[:idx+1]
	t.offset += int64(len(complete))
	return splitLines(complete), nil
}

func splitLines(data []byte) []string {
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// logFiles returns the .log file paths under dir sorted by modification time.
func logFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	type fileInfo struct {
		path string
		mod  int64
	}
	files := make([]fileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: filepath.Join(dir, entry.Name()), mod: info.ModTime().UnixNano()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod < files[j].mod })

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.path
	}
	return paths, nil
}
