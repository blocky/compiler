package logsniffer

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"
)

func NewLogger() *Logger {
	rec := newSnifferWithHandler(slog.Default().Handler())
	return &Logger{
		rec: rec,
		log: slog.New(rec),
	}
}

func NewSilentLogger() *Logger {
	rec := newSniffer()
	return &Logger{
		rec: rec,
		log: slog.New(rec),
	}
}

type Logger struct {
	log *slog.Logger
	rec *sniffer
}

func (l *Logger) Slog() *slog.Logger {
	return l.log
}

func msgContains(record slog.Record, phrase string) bool {
	return strings.Contains(record.Message, phrase)
}

func attrsContain(record slog.Record, phrase string) bool {
	phraseFound := false
	record.Attrs(func(a slog.Attr) bool {
		if strings.Contains(a.Value.String(), phrase) {
			phraseFound = true
			return false
		}
		return true
	})
	return phraseFound
}

func (l *Logger) LevelCount(level slog.Level) int {
	l.rec.Lock()
	defer l.rec.Unlock()
	return len(l.rec.records[level])
}

func (l *Logger) Records(level slog.Level) []slog.Record {
	l.rec.Lock()
	defer l.rec.Unlock()
	return l.rec.records[level]
}

func sortAttrsByKey(attrs []slog.Attr) []slog.Attr {
	sortedAttrs := make([]slog.Attr, len(attrs))
	copy(sortedAttrs, attrs)
	sort.SliceStable(sortedAttrs, func(i, j int) bool {
		return sortedAttrs[i].Key < sortedAttrs[j].Key
	})
	return sortedAttrs
}

func attrsEqual(one, two []slog.Attr) bool {
	if len(one) != len(two) {
		return false
	}

	sortedOne := sortAttrsByKey(one)
	sortedTwo := sortAttrsByKey(two)

	for i := range sortedOne {
		if !sortedOne[i].Equal(sortedTwo[i]) {
			return false
		}
	}
	return true
}

func allAttrs(r slog.Record) []slog.Attr {
	var all []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		all = append(all, slog.Attr{Key: a.Key, Value: a.Value})
		return true
	})
	return all
}

func (l *Logger) RecordCountWithAttrs(
	level slog.Level,
	msg string,
	attrs []slog.Attr,
) int {
	l.rec.Lock()
	defer l.rec.Unlock()
	matched := 0
	for _, record := range l.rec.records[level] {
		if record.Message != msg {
			continue
		}
		all := allAttrs(record)
		if attrsEqual(all, attrs) {
			matched++
		}
	}
	return matched
}

func (l *Logger) RecordCount(
	level slog.Level,
	msg string,
) int {
	l.rec.Lock()
	defer l.rec.Unlock()

	matched := 0
	for _, record := range l.rec.records[level] {
		if record.Message == msg {
			matched++
		}
	}
	return matched
}

func (l *Logger) Logged(level slog.Level, phrase string) bool {
	l.rec.Lock()
	defer l.rec.Unlock()

	for _, record := range l.rec.records[level] {
		if msgContains(record, phrase) {
			return true
		}
		if attrsContain(record, phrase) {
			return true
		}
	}
	return false
}

func newSnifferWithHandler(baseHandler slog.Handler) *sniffer {
	return &sniffer{
		records:     make(map[slog.Level][]slog.Record),
		baseHandler: baseHandler,
	}
}

func newSniffer() *sniffer {
	return &sniffer{
		records:     make(map[slog.Level][]slog.Record),
		baseHandler: nil,
	}
}

type sniffer struct {
	baseHandler slog.Handler
	records     map[slog.Level][]slog.Record
	mutex       sync.Mutex
}

func (s *sniffer) Lock() {
	s.mutex.Lock()
}

func (s *sniffer) Unlock() {
	s.mutex.Unlock()
}

func (s *sniffer) Handle(ctx context.Context, record slog.Record) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	cloned := slog.Record{
		Message: record.Message,
	}

	all := allAttrs(record)
	if len(all) > 0 {
		cloned.AddAttrs(all...)
	}
	s.records[record.Level] = append(s.records[record.Level], cloned)
	if s.baseHandler != nil {
		return s.baseHandler.Handle(ctx, record)
	}
	return nil
}

func (s *sniffer) Enabled(_ context.Context, level slog.Level) bool { return true }
func (s *sniffer) WithAttrs(attrs []slog.Attr) slog.Handler         { return s }
func (s *sniffer) WithGroup(name string) slog.Handler               { return s }
