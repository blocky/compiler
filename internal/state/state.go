package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/google/uuid"
)

type State struct {
	dir  string
	time time.Time

	ids []string
}

func InstanceDir(dirPath string, pid int, id uuid.UUID) string {
	instanceDirName := fmt.Sprintf("%d.%s", pid, id)
	return filepath.Join(dirPath, instanceDirName)
}

// todo: when creating use xdg.StateHome+appName as dirPath
func Init(dirPath string, pID int) (*State, error) {
	s := &State{
		dir:  InstanceDir(dirPath, pID, uuid.New()),
		time: time.Now(),
		ids:  make([]string, 0),
	}
	if err := s.persist(); err != nil {
		return nil, fmt.Errorf("persisting new state: %w", err)
	}
	return s, nil
}

func (s *State) writeMetadata() error {
	dst := filepath.Join(s.dir, "metadata.json")
	metadata := struct {
		Time string `json:"time"`
	}{
		Time: s.time.Format(time.RFC3339),
	}
	marshaled, err := json.MarshalIndent(&metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	if err = os.WriteFile(dst, marshaled, 0600); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}
	return nil
}

func (s *State) writeIDs() error {
	dst := filepath.Join(s.dir, "ids.json")
	marshaled, err := json.Marshal(&s.ids)
	if err != nil {
		return fmt.Errorf("marshaling ids: %w", err)
	}
	if err = os.WriteFile(dst, marshaled, 0600); err != nil {
		return fmt.Errorf("writing ids: %w", err)
	}
	return nil
}

func (s *State) persist() error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}
	if err := s.writeMetadata(); err != nil {
		return fmt.Errorf("persisting metadata: %w", err)
	}
	if err := s.writeIDs(); err != nil {
		return fmt.Errorf("persisting ids: %w", err)
	}
	return nil
}

func (s *State) CleanUp() error {
	if err := os.RemoveAll(s.dir); err != nil {
		return fmt.Errorf("removing state directory: %w", err)
	}
	return nil
}

func (s *State) AddID(id string) error {
	s.ids = append(s.ids, id)
	return s.writeIDs()
}

func (s *State) RemoveID(id string) error {
	s.ids = slices.DeleteFunc(s.ids, func(curr string) bool {
		return curr == id
	})
	return s.writeIDs()
}

func (s *State) IDs() []string {
	return s.ids
}

func (s *State) Dir() string {
	return s.dir
}

func (s *State) Time() time.Time {
	return s.time
}
