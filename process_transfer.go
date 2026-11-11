package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"prochub/internal/logging"
	"prochub/internal/process"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// transferFormat identifies files written by ExportProcesses.
	transferFormat = "prochub.processes"
	// transferVersion is bumped when the envelope changes incompatibly. Older
	// files are still accepted because decoding only needs "processes".
	transferVersion = 1

	// importModeSkip keeps the existing process when a name is already used.
	importModeSkip = "skip"
	// importModeOverwrite replaces the existing same-name definition while
	// keeping its ID and runtime state.
	importModeOverwrite = "overwrite"
)

// Validation codes reported in ProcessImportCandidate.Error. The UI maps them
// to localized text, so they stay stable and language-neutral.
const (
	importErrorNameRequired    = "name_required"
	importErrorCommandRequired = "command_required"
)

var (
	// errNoProcessesToExport is returned when there is nothing to write.
	errNoProcessesToExport = errors.New("no processes to export")
	// errInvalidProcessFile is wrapped by every parse/validation failure so the
	// UI can map it to a single "invalid file" message.
	errInvalidProcessFile = errors.New("invalid process file")
)

// ProcessTransferFile is the versioned envelope written by ExportProcesses and
// read by ImportProcesses. Only definitions are transferred: runtime state
// (PID, restart count, logs) belongs to the machine that produced it.
type ProcessTransferFile struct {
	Format     string               `json:"format"`
	Version    int                  `json:"version"`
	App        string               `json:"app,omitempty"`
	AppVersion string               `json:"appVersion,omitempty"`
	ExportedAt time.Time            `json:"exportedAt"`
	Processes  []process.Definition `json:"processes"`
}

// ProcessImportCandidate is one entry of an import preview. Conflict is set to
// "name" when the name is already used by an existing process or by an earlier
// entry of the same file; Error is set when the entry cannot be imported at all
// (missing name or command).
type ProcessImportCandidate struct {
	Definition process.Definition `json:"definition"`
	Conflict   string             `json:"conflict"`
	Error      string             `json:"error"`
}

// ProcessImportPreview describes a process file before anything is applied, so
// the user can review the entries and choose how to treat same-name conflicts.
type ProcessImportPreview struct {
	Path       string                   `json:"path"`
	AppVersion string                   `json:"appVersion"`
	ExportedAt *time.Time               `json:"exportedAt,omitempty"`
	Total      int                      `json:"total"`
	Importable int                      `json:"importable"`
	Conflicts  int                      `json:"conflicts"`
	Processes  []ProcessImportCandidate `json:"processes"`
}

// ProcessImportResult reports what ImportProcesses actually changed.
type ProcessImportResult struct {
	Total    int      `json:"total"`
	Imported []string `json:"imported"`
	Updated  []string `json:"updated"`
	Skipped  []string `json:"skipped"`
	Invalid  []string `json:"invalid"`
}

// ExportProcesses writes every process definition to a user-chosen JSON file
// and returns its path, or "" when the user cancelled the dialog.
func (a *App) ExportProcesses() (string, error) {
	defer a.recoverPanic("ExportProcesses")

	a.processMu.Lock()
	defs := make([]process.Definition, 0, len(a.config.Processes))
	for _, def := range a.config.Processes {
		defs = append(defs, cloneDefinition(def))
	}
	a.processMu.Unlock()

	if len(defs) == 0 {
		return "", errNoProcessesToExport
	}

	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Processes",
		DefaultFilename: fmt.Sprintf("prochub-processes-%s.json", time.Now().Format("2006-01-02")),
		Filters: []runtime.FileFilter{
			{DisplayName: "ProcHub Process Files", Pattern: "*.json"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		a.LogSystemError("ExportProcesses", fmt.Sprintf("Failed to open save dialog: %v", err))
		return "", err
	}
	if filePath == "" {
		return "", nil // user cancelled
	}
	if !strings.HasSuffix(strings.ToLower(filePath), ".json") {
		filePath += ".json"
	}

	payload := buildTransferFile(defs, appConfig.Version, time.Now())
	if err := writeProcessTransferFile(filePath, payload); err != nil {
		a.LogSystemError("ExportProcesses", fmt.Sprintf("Failed to write %s: %v", filePath, err))
		return "", err
	}
	return filePath, nil
}

// SelectProcessImportFile opens the file dialog used by the import flow and
// returns the chosen path, or "" when the user cancelled.
func (a *App) SelectProcessImportFile() (string, error) {
	defer a.recoverPanic("SelectProcessImportFile")

	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import Processes",
		Filters: []runtime.FileFilter{
			{DisplayName: "ProcHub Process Files", Pattern: "*.json"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		a.LogSystemError("SelectProcessImportFile", fmt.Sprintf("Failed to open file dialog: %v", err))
		return "", err
	}
	return filePath, nil
}

// PreviewProcessImport parses a file and reports the entries and conflicts
// without changing any configuration.
func (a *App) PreviewProcessImport(path string) (ProcessImportPreview, error) {
	defer a.recoverPanic("PreviewProcessImport")

	file, err := readProcessTransferFile(path)
	if err != nil {
		a.LogSystemError("PreviewProcessImport", fmt.Sprintf("Failed to read %s: %v", path, err))
		return ProcessImportPreview{}, err
	}

	a.processMu.Lock()
	existing := append([]process.Definition(nil), a.config.Processes...)
	a.processMu.Unlock()

	return buildImportPreview(path, file, existing), nil
}

// ImportProcesses applies a process file. mode is "skip" (keep existing
// same-name processes) or "overwrite" (replace their definitions, keeping the
// existing ID and runtime state). Imported processes get a fresh ID whenever
// the exported one is already taken.
func (a *App) ImportProcesses(path string, mode string) (ProcessImportResult, error) {
	defer a.recoverPanic("ImportProcesses")

	result := ProcessImportResult{
		Imported: []string{},
		Updated:  []string{},
		Skipped:  []string{},
		Invalid:  []string{},
	}

	file, err := readProcessTransferFile(path)
	if err != nil {
		a.LogSystemError("ImportProcesses", fmt.Sprintf("Failed to read %s: %v", path, err))
		return result, err
	}
	result.Total = len(file.Processes)

	if mode != importModeOverwrite {
		mode = importModeSkip
	}

	logDir := a.config.LogDir
	if logDir == "" {
		logDir = "logs"
	}
	logDir = filepath.Join(a.dataDir, logDir)

	a.processMu.Lock()
	cfg := a.config
	nameIndex := indexByName(cfg.Processes)
	usedIDs := make(map[string]bool, len(cfg.Processes))
	for _, def := range cfg.Processes {
		usedIDs[def.ID] = true
	}

	for _, raw := range file.Processes {
		def, normalizeErr := normalizeImportedDefinition(raw)
		if normalizeErr != nil {
			result.Invalid = append(result.Invalid, transferEntryLabel(raw))
			continue
		}

		key := strings.ToLower(def.Name)
		if idx, exists := nameIndex[key]; exists {
			if mode == importModeSkip {
				result.Skipped = append(result.Skipped, cfg.Processes[idx].Name)
				continue
			}
			// Overwrite keeps the existing ID and runtime state: a running
			// process must not be killed by importing configuration. It picks
			// up the new definition on its next start/restart.
			def.ID = cfg.Processes[idx].ID
			cfg.Processes[idx] = def
			if updateErr := a.pm.UpdateDefinition(def.ID, def); updateErr != nil {
				a.pm.Register(def)
			}
			result.Updated = append(result.Updated, def.Name)
			continue
		}

		assignUniqueID(&def, usedIDs)
		cfg.Processes = append(cfg.Processes, def)
		nameIndex[key] = len(cfg.Processes) - 1
		result.Imported = append(result.Imported, def.Name)

		a.pm.Register(def)
		a.loggers[def.ID] = &ProcessLogger{
			store: logging.NewRollingStore(
				filepath.Join(logDir, def.ID),
				a.config.MaxLogLines,
				a.config.MaxLogFiles,
			),
			hub: logging.NewStreamHub(100),
		}
	}
	a.processMu.Unlock()

	// Nothing was changed (all skipped/invalid): keep the persisted config as is.
	if len(result.Imported)+len(result.Updated) == 0 {
		return result, nil
	}

	if err := a.applyConfig(cfg, originGUI); err != nil {
		a.LogSystemError("ImportProcesses", fmt.Sprintf("Failed to save config after import: %v", err))
		return result, err
	}
	a.notifyProcessesChanged()
	return result, nil
}

// buildTransferFile wraps definitions in the export envelope.
func buildTransferFile(defs []process.Definition, appVersion string, now time.Time) ProcessTransferFile {
	cloned := make([]process.Definition, 0, len(defs))
	for _, def := range defs {
		cloned = append(cloned, cloneDefinition(def))
	}
	return ProcessTransferFile{
		Format:     transferFormat,
		Version:    transferVersion,
		App:        AppName,
		AppVersion: appVersion,
		ExportedAt: now,
		Processes:  cloned,
	}
}

// writeProcessTransferFile marshals the envelope with indentation so the file
// stays readable and diffable.
func writeProcessTransferFile(path string, file ProcessTransferFile) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// readProcessTransferFile reads and parses a file from disk.
func readProcessTransferFile(path string) (ProcessTransferFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProcessTransferFile{}, err
	}
	return parseProcessTransferFile(data)
}

// parseProcessTransferFile accepts three shapes so users can migrate from
// older data:
//   - the export envelope written by ExportProcesses;
//   - a bare JSON array of process definitions;
//   - a full config.json, whose "processes" field is enough.
func parseProcessTransferFile(data []byte) (ProcessTransferFile, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return ProcessTransferFile{}, fmt.Errorf("%w: file is empty", errInvalidProcessFile)
	}

	if trimmed[0] == '[' {
		var defs []process.Definition
		if err := json.Unmarshal(trimmed, &defs); err != nil {
			return ProcessTransferFile{}, fmt.Errorf("%w: %v", errInvalidProcessFile, err)
		}
		return ProcessTransferFile{
			Format:     transferFormat,
			Version:    transferVersion,
			Processes:  defs,
			ExportedAt: time.Time{},
		}, nil
	}

	var file ProcessTransferFile
	if err := json.Unmarshal(trimmed, &file); err != nil {
		return ProcessTransferFile{}, fmt.Errorf("%w: %v", errInvalidProcessFile, err)
	}
	if file.Format != "" && file.Format != transferFormat {
		return ProcessTransferFile{}, fmt.Errorf("%w: unknown format %q", errInvalidProcessFile, file.Format)
	}
	if file.Version > transferVersion {
		return ProcessTransferFile{}, fmt.Errorf(
			"%w: file version %d is newer than supported version %d",
			errInvalidProcessFile, file.Version, transferVersion,
		)
	}
	if file.Processes == nil {
		return ProcessTransferFile{}, fmt.Errorf("%w: no processes found", errInvalidProcessFile)
	}
	return file, nil
}

// buildImportPreview analyses a parsed file against the current definitions.
func buildImportPreview(path string, file ProcessTransferFile, existing []process.Definition) ProcessImportPreview {
	preview := ProcessImportPreview{
		Path:       path,
		AppVersion: file.AppVersion,
		Total:      len(file.Processes),
		Processes:  make([]ProcessImportCandidate, 0, len(file.Processes)),
	}
	if !file.ExportedAt.IsZero() {
		exportedAt := file.ExportedAt
		preview.ExportedAt = &exportedAt
	}

	existingNames := indexByName(existing)
	seenNames := make(map[string]bool, len(file.Processes))
	for _, raw := range file.Processes {
		candidate := ProcessImportCandidate{Definition: cloneDefinition(raw)}
		def, err := normalizeImportedDefinition(raw)
		if err != nil {
			candidate.Error = err.Error()
			preview.Processes = append(preview.Processes, candidate)
			continue
		}
		candidate.Definition = def

		// A name clash is either an existing process or an earlier entry of the
		// same file; both follow the same skip/overwrite choice.
		key := strings.ToLower(def.Name)
		_, exists := existingNames[key]
		duplicate := seenNames[key]
		seenNames[key] = true
		if exists || duplicate {
			candidate.Conflict = "name"
			preview.Conflicts++
		}
		preview.Importable++
		preview.Processes = append(preview.Processes, candidate)
	}
	return preview
}

// normalizeImportedDefinition trims and validates one imported definition,
// applying the same defaults as AddProcess so hand-written files work too.
func normalizeImportedDefinition(def process.Definition) (process.Definition, error) {
	def.Name = strings.TrimSpace(def.Name)
	def.Command = strings.TrimSpace(def.Command)
	def.WorkingDir = strings.TrimSpace(def.WorkingDir)
	def.ID = strings.TrimSpace(def.ID)

	if def.Name == "" {
		return def, errors.New(importErrorNameRequired)
	}
	if def.Command == "" {
		return def, errors.New(importErrorCommandRequired)
	}
	if def.RestartPolicy != process.RestartAlways && def.RestartPolicy != process.RestartNever {
		def.RestartPolicy = process.RestartOnFailure
	}
	if def.MaxRetries < 0 {
		def.MaxRetries = 0
	}
	if def.Args == nil {
		def.Args = []string{}
	}
	if def.Env == nil {
		def.Env = process.Environment{}
	}
	return def, nil
}

// transferEntryLabel names an invalid entry in the import result.
func transferEntryLabel(def process.Definition) string {
	if name := strings.TrimSpace(def.Name); name != "" {
		return name
	}
	if command := strings.TrimSpace(def.Command); command != "" {
		return command
	}
	return "?"
}

// assignUniqueID keeps a free exported ID and otherwise allocates a new one,
// so a definition whose ID is already taken never hijacks an unrelated process.
func assignUniqueID(def *process.Definition, used map[string]bool) {
	if def.ID != "" && !used[def.ID] {
		used[def.ID] = true
		return
	}
	for i := 1; ; i++ {
		id := fmt.Sprintf("proc-%d", i)
		if !used[id] {
			def.ID = id
			used[id] = true
			return
		}
	}
}

// indexByName maps lower-cased process names to their index. Names are the
// identity users recognise, so imports match on them regardless of ID.
func indexByName(defs []process.Definition) map[string]int {
	index := make(map[string]int, len(defs))
	for i, def := range defs {
		name := strings.ToLower(strings.TrimSpace(def.Name))
		if name == "" {
			continue
		}
		index[name] = i
	}
	return index
}

// cloneDefinition deep-copies the slices and map so later edits cannot alias
// the live configuration.
func cloneDefinition(def process.Definition) process.Definition {
	clone := def
	if def.Args != nil {
		clone.Args = append([]string(nil), def.Args...)
	}
	if def.Env != nil {
		clone.Env = make(process.Environment, len(def.Env))
		for k, v := range def.Env {
			clone.Env[k] = v
		}
	}
	return clone
}
