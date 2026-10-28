// Package store saves the club file beside the program so the folder stays portable.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"pitchboard/internal/model"
)

const FileName = "pitchboard.json"

type Store struct {
	mu   sync.RWMutex
	path string
	db   model.Database
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, FileName)
	s := &Store{path: path}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.db = fresh()
		if err := s.persist(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &s.db); err != nil {
		return nil, errors.New("the Pitchboard data file could not be read; a copy may be in pitchboard.json.bak")
	}
	if s.db.Version != 0 && s.db.Version != model.Version {
		return nil, errors.New("this data file was written by a newer Pitchboard")
	}
	s.db.Normalize()
	EnsureCatalogs(&s.db)
	s.db.Normalize()
	return s, nil
}

func fresh() model.Database {
	db := model.Database{
		Version:    model.Version,
		Settings:   model.DefaultSettings(timeNow()),
		AgeGroups:  []model.AgeGroup{},
		Teams:      []model.Team{},
		Venues:     []model.Venue{},
		Pitches:    []model.Pitch{},
		Surfaces:   []model.Surface{},
		Formats:    []model.Format{},
		EventTypes: []model.EventType{},
		Events:     []model.Event{},
		Blackouts:  []model.Blackout{},
	}
	EnsureCatalogs(&db)
	db.Normalize()
	return db
}

func (s *Store) Path() string { return s.path }

func (s *Store) Dir() string { return filepath.Dir(s.path) }

func (s *Store) View() model.Database {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clone(s.db)
}

func (s *Store) Update(fn func(*model.Database) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.db)
	if err := fn(&next); err != nil {
		return err
	}
	next.Normalize()
	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := atomicWrite(s.path, raw); err != nil {
		return err
	}
	s.db = next
	return nil
}

func (s *Store) persist() error {
	s.db.Normalize()
	raw, err := json.MarshalIndent(s.db, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return atomicWrite(s.path, raw)
}

func clone(db model.Database) model.Database {
	raw, err := json.Marshal(db)
	if err != nil {
		return db
	}
	var out model.Database
	if err := json.Unmarshal(raw, &out); err != nil {
		return db
	}
	out.Normalize()
	return out
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		bak := path + ".bak"
		_ = os.Remove(bak)
		if err := os.Rename(path, bak); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Rename(path+".bak", path)
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
