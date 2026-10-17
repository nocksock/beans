package sessionpoller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/launcher"
)

const (
	DefaultPollIntervalMs = 200
	DefaultCheckTimeoutMs = 500
)

type SessionStatus struct {
	HasSession bool
	Error      string
	LastCheck  time.Time
}

type Poller struct {
	config    *config.SessionConfig
	beansRoot string
	inProgIDs []string

	mu       sync.RWMutex
	statuses map[string]SessionStatus

	ticker *time.Ticker
	done   chan struct{}
	index  int
}

func New(cfg *config.SessionConfig, beansRoot string) *Poller {
	if cfg == nil {
		cfg = &config.SessionConfig{}
	}
	if cfg.PollIntervalMs <= 0 {
		cfg.PollIntervalMs = DefaultPollIntervalMs
	}
	if cfg.CheckTimeoutMs <= 0 {
		cfg.CheckTimeoutMs = DefaultCheckTimeoutMs
	}

	return &Poller{
		config:    cfg,
		beansRoot: beansRoot,
		statuses:  make(map[string]SessionStatus),
		done:      make(chan struct{}),
	}
}

func (p *Poller) Start(beans []*bean.Bean) {
	p.mu.Lock()
	p.inProgIDs = p.filterInProgress(beans)
	p.mu.Unlock()

	if p.config == nil || p.config.Check == "" {
		return
	}

	p.ticker = time.NewTicker(time.Duration(p.config.PollIntervalMs) * time.Millisecond)
	go p.run()
}

func (p *Poller) Stop() {
	if p.ticker != nil {
		p.ticker.Stop()
	}
	close(p.done)
}

func (p *Poller) run() {
	for {
		select {
		case <-p.done:
			return
		case <-p.ticker.C:
			p.tick()
		}
	}
}

func (p *Poller) tick() {
	p.mu.RLock()
	ids := p.inProgIDs
	p.mu.RUnlock()

	if len(ids) == 0 {
		return
	}

	p.mu.Lock()
	p.index = p.index % len(ids)
	beanID := ids[p.index]
	p.mu.Unlock()

	status := p.checkSession(beanID)

	p.mu.Lock()
	p.statuses[beanID] = status
	p.mu.Unlock()
}

func (p *Poller) checkSession(beanID string) SessionStatus {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.config.CheckTimeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", p.config.Check)
	cmd.Dir = p.config.CheckDir
	cmd.Env = []string{
		fmt.Sprintf("BEANS_CONFIG_DIR=%s", p.config.CheckDir),
		fmt.Sprintf("BEANS_ROOT=%s", p.beansRoot),
		fmt.Sprintf("BEANS_DIR=%s", p.beansRoot),
		fmt.Sprintf("BEANS_ID=%s", beanID),
	}

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return SessionStatus{Error: "timeout"}
		}
		return SessionStatus{HasSession: false}
	}
	return SessionStatus{HasSession: true, LastCheck: time.Now()}
}

func (p *Poller) GetStatus(beanID string) SessionStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.statuses[beanID]
}

func (p *Poller) GetAllStatuses() map[string]SessionStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]SessionStatus, len(p.statuses))
	for k, v := range p.statuses {
		result[k] = v
	}
	return result
}

func (p *Poller) RefreshBeans(beans []*bean.Bean) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.inProgIDs = p.filterInProgress(beans)
	if p.index >= len(p.inProgIDs) {
		p.index = 0
	}
}

func (p *Poller) filterInProgress(beans []*bean.Bean) []string {
	var ids []string
	for _, b := range beans {
		if b.Status == "in-progress" {
			ids = append(ids, b.ID)
		}
	}
	return ids
}

func (p *Poller) GetInProgressIDs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ids := make([]string, len(p.inProgIDs))
	copy(ids, p.inProgIDs)
	return ids
}

func (p *Poller) HasActiveSessions() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, s := range p.statuses {
		if s.HasSession {
			return true
		}
	}
	return false
}

// CreateExecCommand creates a command that runs a session script via sh -c.
func CreateExecCommand(l launcher.Launch) (*exec.Cmd, error) {
	cmd := exec.Command("sh", "-c", l.Script)
	cmd.Env = l.Env()
	cmd.Dir = l.WorkDir()

	return cmd, nil
}

// CreateLoggedExecCommand is like CreateExecCommand, but additionally tees the
// command's stderr into a temporary log file. Stdin and stdout are left unset so
// tea.ExecProcess attaches them to the terminal and interactive launchers keep
// working. The caller must close the returned file after the command exits.
func CreateLoggedExecCommand(l launcher.Launch) (*exec.Cmd, *os.File, error) {
	cmd, err := CreateExecCommand(l)
	if err != nil {
		return nil, nil, err
	}

	logFile, err := os.CreateTemp("", "beans-launch-*.log")
	if err != nil {
		return nil, nil, err
	}
	cmd.Stderr = io.MultiWriter(os.Stderr, logFile)

	return cmd, logFile, nil
}

// ExitCode extracts the process exit code from the error returned by running a
// command. It returns 0 for a nil error and -1 if the command could not be run.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
