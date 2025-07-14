package logsniffer_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/blocky/compiler/internal/logsniffer"
)

func TestNewLogger(t *testing.T) {
	// when
	got := logsniffer.NewLogger()

	// then
	assert.NotNil(t, got)
}

func TestNewLSilentLogger(t *testing.T) {
	// when
	got := logsniffer.NewSilentLogger()

	// then
	assert.NotNil(t, got)
}

func TestLogger_Slog(t *testing.T) {
	// given
	sut := logsniffer.NewSilentLogger()

	// when
	got := sut.Slog()

	// then
	assert.NotNil(t, got)
}

func TestLogger_LevelCount(t *testing.T) {
	// given
	sut := logsniffer.NewSilentLogger()

	wantDebugCount := 4
	for i := 0; i < wantDebugCount; i++ {
		sut.Slog().Debug("debug msg")
	}
	wantInfoCount := 3
	for i := 0; i < wantInfoCount; i++ {
		sut.Slog().Info("info msg")
	}
	wantWarnCount := 2
	for i := 0; i < wantWarnCount; i++ {
		sut.Slog().Warn("warn msg")
	}
	wantErrorCount := 1
	for i := 0; i < wantErrorCount; i++ {
		sut.Slog().Error("error msg")
	}

	// when & then
	assert.Equal(t, wantDebugCount, sut.LevelCount(slog.LevelDebug))
	assert.Equal(t, wantInfoCount, sut.LevelCount(slog.LevelInfo))
	assert.Equal(t, wantWarnCount, sut.LevelCount(slog.LevelWarn))
	assert.Equal(t, wantErrorCount, sut.LevelCount(slog.LevelError))
}

func TestLogger_Records(t *testing.T) {
	// given
	sut := logsniffer.NewSilentLogger()

	wantDebugCount := 4
	for i := 0; i < wantDebugCount; i++ {
		sut.Slog().Debug("debug msg")
	}
	wantInfoCount := 3
	for i := 0; i < wantInfoCount; i++ {
		sut.Slog().Info("info msg")
	}
	wantWarnCount := 2
	for i := 0; i < wantWarnCount; i++ {
		sut.Slog().Warn("warn msg")
	}
	wantErrorCount := 1
	for i := 0; i < wantErrorCount; i++ {
		sut.Slog().Error("error msg")
	}

	// when & then
	assert.Equal(t, wantDebugCount, len(sut.Records(slog.LevelDebug)))
	assert.Equal(t, wantInfoCount, len(sut.Records(slog.LevelInfo)))
	assert.Equal(t, wantWarnCount, len(sut.Records(slog.LevelWarn)))
	assert.Equal(t, wantErrorCount, len(sut.Records(slog.LevelError)))
}

func logOnAllLevels(log *slog.Logger) {
	log.Error("my error message", slog.String("error-attr-key", "error-attr-value"))
	log.Info("my info message", slog.String("info-attr-key", "info-attr-value"))
	log.Warn("my warn message", slog.String("warn-attr-key", "warn-attr-value"))
	log.Debug("my debug message", slog.String("debug-attr-key", "debug-attr-value"))
}

func TestLogger_RecordCountWithAttrs(t *testing.T) {
	for name, tc := range map[string]struct {
		wantCount          int
		wantCountWithAttrs int
		wantLevel          slog.Level
	}{
		"happy path - one record without attrs": {
			wantCount:          1,
			wantCountWithAttrs: 0,
			wantLevel:          slog.LevelInfo,
		},
		"happy path - one record with attrs": {
			wantCount:          0,
			wantCountWithAttrs: 1,
			wantLevel:          slog.LevelDebug,
		},
		"happy path - multiple records": {
			wantCount:          10,
			wantCountWithAttrs: 5,
			wantLevel:          slog.LevelWarn,
		},
		"happy path - no records": {
			wantCount:          0,
			wantCountWithAttrs: 0,
			wantLevel:          slog.LevelError,
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			ctx := context.Background()
			wantMsg := "my want message"
			wantAttrs := []slog.Attr{
				slog.String("key1", "val1"),
				slog.String("key2", "val2"),
			}
			sut := logsniffer.NewSilentLogger()

			logOnAllLevels(sut.Slog())

			for i := 0; i < tc.wantCount; i++ {
				sut.Slog().LogAttrs(ctx, tc.wantLevel, wantMsg, wantAttrs...)
			}

			for i := 0; i < tc.wantCountWithAttrs; i++ {
				sut.Slog().Log(ctx, tc.wantLevel, wantMsg)
			}

			// when
			gotCount := sut.RecordCountWithAttrs(tc.wantLevel, wantMsg, wantAttrs)

			// then
			assert.Equal(t, tc.wantCount, gotCount)
		})
	}
}

func TestLogger_RecordCount(t *testing.T) {
	for name, tc := range map[string]struct {
		wantCount          int
		wantCountWithAttrs int
		wantLevel          slog.Level
	}{
		"happy path - one record without attrs": {
			wantCount:          1,
			wantCountWithAttrs: 0,
			wantLevel:          slog.LevelInfo,
		},
		"happy path - one record with attrs": {
			wantCount:          0,
			wantCountWithAttrs: 1,
			wantLevel:          slog.LevelDebug,
		},
		"happy path - multiple records": {
			wantCount:          10,
			wantCountWithAttrs: 5,
			wantLevel:          slog.LevelWarn,
		},
		"happy path - no records": {
			wantCount:          0,
			wantCountWithAttrs: 0,
			wantLevel:          slog.LevelError,
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			ctx := context.Background()
			wantMsg := "my want message"
			wantAttrs := []slog.Attr{
				slog.String("key1", "val1"),
				slog.String("key2", "val2"),
			}
			sut := logsniffer.NewSilentLogger()

			logOnAllLevels(sut.Slog())

			for i := 0; i < tc.wantCount; i++ {
				sut.Slog().LogAttrs(ctx, tc.wantLevel, wantMsg, wantAttrs...)
			}

			for i := 0; i < tc.wantCountWithAttrs; i++ {
				sut.Slog().Log(ctx, tc.wantLevel, wantMsg)
			}

			// when
			gotCount := sut.RecordCount(tc.wantLevel, wantMsg)

			// then
			assert.Equal(t, tc.wantCount+tc.wantCountWithAttrs, gotCount)
		})
	}
}

func TestLogger_Logged(t *testing.T) {
	t.Run("happy path - msg was logged", func(t *testing.T) {
		// given
		wantPhrase := "i search for that string"
		wantLevel := slog.LevelInfo

		sut := logsniffer.NewSilentLogger()
		sut.Slog().Log(context.Background(), wantLevel, "log happened "+wantPhrase)

		// when
		gotFound := sut.Logged(wantLevel, wantPhrase)

		// then
		assert.True(t, gotFound, "could not find expected phrase '%s'", wantPhrase)
	})

	t.Run("happy path - msg not found on given level", func(t *testing.T) {
		// given
		wantPhrase := "i search for that string"
		wantLevel := slog.LevelInfo

		sut := logsniffer.NewSilentLogger()
		sut.Slog().Log(context.Background(), wantLevel, "log happened "+wantPhrase)

		// when
		gotFound := sut.Logged(slog.LevelWarn, wantPhrase)

		// then
		assert.False(t, gotFound, "unexpected phrase '%s'", wantPhrase)
	})

	t.Run("happy path - attr value was logged", func(t *testing.T) {
		// given
		wantPhrase := "i search for that string"
		wantLevel := slog.LevelInfo

		sut := logsniffer.NewSilentLogger()
		sut.Slog().Log(
			context.Background(),
			wantLevel, "log happened",
			slog.String("phrase", wantPhrase),
		)

		// when
		gotFound := sut.Logged(wantLevel, wantPhrase)

		// then
		assert.True(t, gotFound, "could not find expected phrase '%s'", wantPhrase)
	})

	t.Run("happy path - attr not found on given level", func(t *testing.T) {
		// given
		wantPhrase := "i search for that string"
		wantLevel := slog.LevelInfo

		sut := logsniffer.NewSilentLogger()
		sut.Slog().Log(
			context.Background(),
			wantLevel, "log happened",
			slog.String("phrase", wantPhrase),
		)

		// when
		gotFound := sut.Logged(slog.LevelWarn, wantPhrase)

		// then
		assert.False(t, gotFound, "unexpected phrase '%s'", wantPhrase)
	})

	t.Run("happy path - searched phrase not found", func(t *testing.T) {
		// given
		wantPhrase := "i search for that string"
		wantLevel := slog.LevelInfo

		sut := logsniffer.NewSilentLogger()
		sut.Slog().Log(
			context.Background(),
			wantLevel,
			"log happened",
			slog.String("phrase", "attrs happened"),
		)

		// when
		gotFound := sut.Logged(wantLevel, wantPhrase)

		// then
		assert.False(t, gotFound, "unexpected phrase '%s'", wantPhrase)
	})
}
