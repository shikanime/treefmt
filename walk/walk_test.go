package walk_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/numtide/treefmt/v2/stats"
	"github.com/numtide/treefmt/v2/test"
	"github.com/numtide/treefmt/v2/walk"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

var sourceExample = `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}`

func TestWatchReader(t *testing.T) {
	as := require.New(t)

	tempDir := test.TempExamples(t)
	statz := stats.New()

	r, err := walk.NewWatchReader(tempDir, "", &statz)
	as.NoError(err)

	eg := errgroup.Group{}
	for _, example := range test.ExamplesPaths {
		eg.Go(func() error {
			filePath := path.Join(tempDir, example)
			content, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			return os.WriteFile(filePath, content, 0o644)
		})
	}

	count := 0

	for count < 33 {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

		files := make([]*walk.File, 8)
		n, err := r.Read(ctx, files)

		count += n

		cancel()

		if errors.Is(err, io.EOF) {
			break
		}
	}

	as.NoError(eg.Wait())

	as.Equal(33, count)
	as.Equal(33, statz.Value(stats.Traversed))
	as.Equal(0, statz.Value(stats.Matched))
	as.Equal(0, statz.Value(stats.Formatted))
	as.Equal(0, statz.Value(stats.Changed))
}

func TestWatchReaderCreate(t *testing.T) {
	as := require.New(t)

	tempDir := t.TempDir()
	statz := stats.New()

	r, err := walk.NewWatchReader(tempDir, "", &statz)
	as.NoError(err)

	as.NoError(
		os.WriteFile(
			path.Join(tempDir, "main.go"),
			[]byte(sourceExample),
			0o644,
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

	files := make([]*walk.File, 8)
	n, err := r.Read(ctx, files)

	cancel()

	if !errors.Is(err, io.EOF) {
		as.NoError(err)
	}

	as.Equal(1, n)
	as.Equal(1, statz.Value(stats.Traversed))
	as.Equal(0, statz.Value(stats.Matched))
	as.Equal(0, statz.Value(stats.Formatted))
	as.Equal(0, statz.Value(stats.Changed))
}
