package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"prochub/internal/process"
)

// TokenHeader is the HTTP header carrying the auth token.
const TokenHeader = "X-ProcHub-Token"

// DefaultPort is the first port tried by the control server.
const DefaultPort = 53093

// MaxPortDrift is how many consecutive ports to try when the base port is busy.
const MaxPortDrift = 10

// maxLogLines caps the number of log lines a single request can fetch.
const maxLogLines = 5000

// StatusInfo is the response of the status endpoint.
type StatusInfo struct {
	Version      string `json:"version"`
	Platform     string `json:"platform"`
	PID          int    `json:"pid"`
	ProcessCount int    `json:"processCount"`
	RunningCount int    `json:"runningCount"`
	DataDir      string `json:"dataDir"`
}

// ConfigSummary is a flattened view of the settings the CLI can read/write.
type ConfigSummary struct {
	Theme        string `json:"theme"`
	Locale       string `json:"locale"`
	AutoStart    bool   `json:"autoStart"`
	AutoStartWSL bool   `json:"autoStartWSL"`
	LogDir       string `json:"logDir"`
}

// ConfigPatch carries optional config updates sent by the CLI. Nil fields are
// left untouched.
type ConfigPatch struct {
	Theme        *string `json:"theme"`
	Locale       *string `json:"locale"`
	AutoStart    *bool   `json:"autoStart"`
	AutoStartWSL *bool   `json:"autoStartWSL"`
}

// WSLStatusInfo is the subset of the WSL runtime state exposed to the CLI.
type WSLStatusInfo struct {
	Supported bool   `json:"supported"`
	Available bool   `json:"available"`
	Running   bool   `json:"running"`
	Starting  bool   `json:"starting"`
	Version   string `json:"version"`
	Distro    string `json:"distro"`
}

// Backend is implemented by the application to serve control requests.
type Backend interface {
	Status() StatusInfo
	ListProcesses() []process.Snapshot
	StartProcess(id string) error
	StopProcess(id string) error
	Logs(id string, tail int) ([]string, error)
	Config() ConfigSummary
	UpdateConfig(patch ConfigPatch) error
	WSLStatus() WSLStatusInfo
	StartWSL() error
	StopWSL() error
	RestartWSL() error
}

// Server is the local HTTP control server embedded in the running application.
type Server struct {
	backend  Backend
	token    string
	basePort int
	port     int
	httpSrv  *http.Server
}

// NewServer creates a control server that tries DefaultPort and a few ports
// above it.
func NewServer(backend Backend, token string) *Server {
	return NewServerWithPort(backend, token, DefaultPort)
}

// NewServerWithPort creates a control server bound to basePort. A basePort of 0
// lets the OS pick a free port (used by tests).
func NewServerWithPort(backend Backend, token string, basePort int) *Server {
	return &Server{backend: backend, token: token, basePort: basePort}
}

// Port returns the bound port. It is only valid after Start.
func (s *Server) Port() int {
	return s.port
}

// Start binds the server on 127.0.0.1 and serves requests in the background.
func (s *Server) Start() (int, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /control/status", s.handleStatus)
	mux.HandleFunc("GET /control/processes", s.handleList)
	mux.HandleFunc("POST /control/processes/{id}/start", s.handleStart)
	mux.HandleFunc("POST /control/processes/{id}/stop", s.handleStop)
	mux.HandleFunc("GET /control/processes/{id}/logs", s.handleLogs)
	mux.HandleFunc("GET /control/config", s.handleConfigGet)
	mux.HandleFunc("POST /control/config", s.handleConfigPatch)
	mux.HandleFunc("GET /control/wsl", s.handleWSLStatus)
	mux.HandleFunc("POST /control/wsl/start", s.handleWSLStart)
	mux.HandleFunc("POST /control/wsl/stop", s.handleWSLStop)
	mux.HandleFunc("POST /control/wsl/restart", s.handleWSLRestart)

	s.httpSrv = &http.Server{
		Handler:           s.withAuth(mux),
		ReadHeaderTimeout: 30 * time.Second,
	}

	ln, err := s.listen()
	if err != nil {
		return 0, err
	}
	s.port = ln.Addr().(*net.TCPAddr).Port
	go func() { _ = s.httpSrv.Serve(ln) }()
	return s.port, nil
}

// Stop shuts the control server down.
func (s *Server) Stop() {
	if s.httpSrv != nil {
		_ = s.httpSrv.Close()
	}
}

func (s *Server) listen() (net.Listener, error) {
	if s.basePort == 0 {
		return net.Listen("tcp", "127.0.0.1:0")
	}
	var (
		ln  net.Listener
		err error
	)
	for i := 0; i < MaxPortDrift; i++ {
		ln, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.basePort+i))
		if err == nil {
			return ln, nil
		}
	}
	return nil, fmt.Errorf("control server listen failed: %w", err)
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(TokenHeader) != s.token {
			writeError(w, http.StatusUnauthorized, "令牌无效")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeData(w, s.backend.Status())
}

func (s *Server) handleList(w http.ResponseWriter, _ *http.Request) {
	items := s.backend.ListProcesses()
	if items == nil {
		items = []process.Snapshot{}
	}
	writeData(w, items)
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.backend.StartProcess(id); err != nil {
		writeError(w, http.StatusBadRequest, describeProcessError(id, err))
		return
	}
	writeData(w, map[string]string{"id": id, "status": "started"})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.backend.StopProcess(id); err != nil {
		writeError(w, http.StatusBadRequest, describeProcessError(id, err))
		return
	}
	writeData(w, map[string]string{"id": id, "status": "stopped"})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tail := 100
	if raw := r.URL.Query().Get("tail"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			tail = n
		}
	}
	if tail > maxLogLines {
		tail = maxLogLines
	}
	lines, err := s.backend.Logs(id, tail)
	if err != nil {
		writeError(w, http.StatusBadRequest, describeProcessError(id, err))
		return
	}
	if lines == nil {
		lines = []string{}
	}
	writeData(w, map[string]interface{}{"lines": lines, "count": len(lines)})
}

func (s *Server) handleConfigGet(w http.ResponseWriter, _ *http.Request) {
	writeData(w, s.backend.Config())
}

func (s *Server) handleConfigPatch(w http.ResponseWriter, r *http.Request) {
	var patch ConfigPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if err := s.backend.UpdateConfig(patch); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeData(w, s.backend.Config())
}

func (s *Server) handleWSLStatus(w http.ResponseWriter, _ *http.Request) {
	writeData(w, s.backend.WSLStatus())
}

func (s *Server) handleWSLStart(w http.ResponseWriter, _ *http.Request) {
	if err := s.backend.StartWSL(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeData(w, s.backend.WSLStatus())
}

func (s *Server) handleWSLStop(w http.ResponseWriter, _ *http.Request) {
	if err := s.backend.StopWSL(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeData(w, s.backend.WSLStatus())
}

func (s *Server) handleWSLRestart(w http.ResponseWriter, _ *http.Request) {
	if err := s.backend.RestartWSL(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeData(w, s.backend.WSLStatus())
}

func describeProcessError(id string, err error) string {
	if errors.Is(err, process.ErrNotFound) {
		return fmt.Sprintf("进程不存在：%s", id)
	}
	return err.Error()
}

func writeData(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "data": data})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": message})
}
