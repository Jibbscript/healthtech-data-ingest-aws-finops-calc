package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	path string
	mu   sync.Mutex
}

type snapshot struct {
	Users       map[string]domain.User       `json:"users"`
	Devices     map[string]domain.Device     `json:"devices"`
	Captures    map[string]domain.Capture    `json:"captures"`
	Findings    map[string]domain.Finding    `json:"findings"`
	AuditEvents map[string]domain.AuditEvent `json:"audit_events"`
}

func New(dataDir string) *Store {
	if dataDir == "" {
		dataDir = ".data"
	}
	return &Store{path: filepath.Join(dataDir, "throne-db.json")}
}

func (s *Store) SeedDemo() error {
	now := time.Now().UTC()
	return s.update(func(ss *snapshot) error {
		ss.Users["user_demo"] = domain.User{ID: "user_demo", Email: "demo@example.com", CreatedAt: now}
		ss.Devices["device_demo"] = domain.Device{ID: "device_demo", UserID: "user_demo", CertificateThumbprint: "DEV-THUMBPRINT", EnrolledAt: now}
		return nil
	})
}

func (s *Store) DeviceAllowedForUser(userID, deviceID, thumbprint string) bool {
	ss, err := s.load()
	if err != nil {
		return false
	}
	d, ok := ss.Devices[deviceID]
	if !ok {
		return false
	}
	return userID != "" && d.UserID == userID && thumbprint != "" && d.CertificateThumbprint == thumbprint
}

func (s *Store) AddCapture(c domain.Capture) (domain.Capture, bool, error) {
	var existing bool
	var out domain.Capture
	err := s.update(func(ss *snapshot) error {
		if c.ID == "" {
			return errors.New("capture id required")
		}
		if cur, ok := ss.Captures[c.ID]; ok {
			existing = true
			out = cur
			return nil
		}
		if c.CapturedAt.IsZero() {
			c.CapturedAt = time.Now().UTC()
		}
		if c.Status == "" {
			c.Status = domain.CapturePending
		}
		ss.Captures[c.ID] = c
		out = c
		return nil
	})
	return out, existing, err
}

func (s *Store) UpdateCaptureStatus(id string, status domain.CaptureStatus, failure string) error {
	return s.update(func(ss *snapshot) error {
		c, ok := ss.Captures[id]
		if !ok {
			return fmt.Errorf("capture %s: %w", id, ErrNotFound)
		}
		c.Status = status
		c.Failure = failure
		ss.Captures[id] = c
		return nil
	})
}

func (s *Store) GetCapture(id string) (domain.Capture, error) {
	ss, err := s.load()
	if err != nil {
		return domain.Capture{}, err
	}
	c, ok := ss.Captures[id]
	if !ok {
		return domain.Capture{}, ErrNotFound
	}
	return c, nil
}

func (s *Store) ListCapturesByUser(userID string, limit int, cursor string) ([]domain.Capture, string, error) {
	ss, err := s.load()
	if err != nil {
		return nil, "", err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	items := make([]domain.Capture, 0)
	for _, c := range ss.Captures {
		if c.UserID == userID {
			items = append(items, c)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CapturedAt.After(items[j].CapturedAt) })
	start := 0
	if cursor != "" {
		for i, c := range items {
			if c.ID == cursor {
				start = i + 1
				break
			}
		}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	next := ""
	if end < len(items) && end > start {
		next = items[end-1].ID
	}
	return items[start:end], next, nil
}

func (s *Store) AddFinding(f domain.Finding) error {
	return s.update(func(ss *snapshot) error {
		if f.ID == "" {
			f.ID = domain.NewID("finding")
		}
		if f.CreatedAt.IsZero() {
			f.CreatedAt = time.Now().UTC()
		}
		ss.Findings[f.ID] = f
		return nil
	})
}

func (s *Store) GetFinding(id string) (domain.Finding, error) {
	ss, err := s.load()
	if err != nil {
		return domain.Finding{}, err
	}
	f, ok := ss.Findings[id]
	if !ok {
		return domain.Finding{}, ErrNotFound
	}
	return f, nil
}

func (s *Store) AddAuditEvent(e domain.AuditEvent) error {
	return s.update(func(ss *snapshot) error {
		if e.ID == "" {
			e.ID = domain.NewID("audit")
		}
		if e.OccurredAt.IsZero() {
			e.OccurredAt = time.Now().UTC()
		}
		ss.AuditEvents[e.ID] = e
		return nil
	})
}

func (s *Store) FindFindingByCapture(captureID string) (domain.Finding, error) {
	ss, err := s.load()
	if err != nil {
		return domain.Finding{}, err
	}
	for _, f := range ss.Findings {
		if f.CaptureID == captureID {
			return f, nil
		}
	}
	return domain.Finding{}, ErrNotFound
}

func (s *Store) update(fn func(*snapshot) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	if err := fn(&ss); err != nil {
		return err
	}
	return s.saveUnlocked(ss)
}

func (s *Store) load() (snapshot, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.loadUnlocked() }

func (s *Store) loadUnlocked() (snapshot, error) {
	ss := snapshot{Users: map[string]domain.User{}, Devices: map[string]domain.Device{}, Captures: map[string]domain.Capture{}, Findings: map[string]domain.Finding{}, AuditEvents: map[string]domain.AuditEvent{}}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return ss, nil
	}
	if err != nil {
		return ss, err
	}
	if len(b) == 0 {
		return ss, nil
	}
	if err := json.Unmarshal(b, &ss); err != nil {
		return ss, err
	}
	if ss.Users == nil {
		ss.Users = map[string]domain.User{}
	}
	if ss.Devices == nil {
		ss.Devices = map[string]domain.Device{}
	}
	if ss.Captures == nil {
		ss.Captures = map[string]domain.Capture{}
	}
	if ss.Findings == nil {
		ss.Findings = map[string]domain.Finding{}
	}
	if ss.AuditEvents == nil {
		ss.AuditEvents = map[string]domain.AuditEvent{}
	}
	return ss, nil
}

func (s *Store) saveUnlocked(ss snapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ss, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}
