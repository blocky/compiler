package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/google/uuid"
)

const NameSeparator = "."

type State struct {
	dir  string
	time time.Time
	log  Logger

	ids []string
}

func InstanceDir(dirPath string, pid int, id uuid.UUID) string {
	instanceDirName := fmt.Sprintf("%d%s%s", pid, NameSeparator, id)
	return filepath.Join(dirPath, instanceDirName)
}

func Init(dirPath string, pID int, log Logger) (*State, error) {
	s := &State{
		dir:  InstanceDir(dirPath, pID, uuid.New()),
		time: time.Now(),
		log:  log,
		ids:  make([]string, 0),
	}
	if err := s.save(); err != nil {
		defer func() {
			_ = s.Remove()
		}()
		return nil, fmt.Errorf("saving new state: %w", err)
	}
	return s, nil
}

func Load(dirPath string, log Logger) (*State, error) {
	s := &State{
		dir: dirPath,
		log: log,
	}
	if err := s.readMetadata(); err != nil {
		return nil, fmt.Errorf("loading metadata: %w", err)
	}
	if err := s.readIDs(); err != nil {
		return nil, fmt.Errorf("loading ids: %w", err)
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

func (s *State) readMetadata() error {
	src := filepath.Join(s.dir, "metadata.json")
	bytes, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading metadata: %w", err)
	}
	metadata := struct {
		Time time.Time `json:"time"`
	}{}
	if err = json.Unmarshal(bytes, &metadata); err != nil {
		return fmt.Errorf("unmarshaling metadata: %w", err)
	}
	s.time = metadata.Time
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

func (s *State) readIDs() error {
	src := filepath.Join(s.dir, "ids.json")
	bytes, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading ids: %w", err)
	}
	var ids []string
	if err := json.Unmarshal(bytes, &ids); err != nil {
		return fmt.Errorf("unmarshaling ids: %w", err)
	}
	s.ids = ids
	return nil
}

func (s *State) save() error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}
	if err := s.writeMetadata(); err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}
	if err := s.writeIDs(); err != nil {
		return fmt.Errorf("saving ids: %w", err)
	}
	return nil
}

func (s *State) Remove() error {
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

func (s *State) CleanIDs(cleaner Cleaner) {
	ctx := context.Background()
	var cleanedIDs []string
	for _, ID := range s.IDs() {
		if err := cleaner.CleanUp(ctx, ID); err != nil {
			s.log.Debug("cleaning up id", "id", ID, "err", err.Error())
			continue
		}
		cleanedIDs = append(cleanedIDs, ID)
	}
	for _, ID := range cleanedIDs {
		if err := s.RemoveID(ID); err != nil {
			s.log.Debug("removing cleaned id", "id", ID, "err", err.Error())
		}
	}
}

func (s *State) Finalize(cleaner Cleaner) error {
	s.CleanIDs(cleaner)
	if len(s.IDs()) == 0 {
		if err := s.Remove(); err != nil {
			s.log.Debug("removing finalized state", "err", err.Error())
		}
	}
	return nil
}
