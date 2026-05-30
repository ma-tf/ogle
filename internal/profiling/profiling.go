// Package profiling provides one-shot pprof dump commands for runtime
// diagnosis of goroutine, heap, and CPU performance.
package profiling

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	tea "charm.land/bubbletea/v2"
)

// ProfilesDumped is delivered by DumpAllCmd with the paths of written profile files.
type ProfilesDumped struct {
	CPUProfilePath string
	GoroutinePath  string
	HeapPath       string
	Err            error
}

// DumpAllCmd returns a Cmd that collects a CPU profile for the given duration
// (if > 0), then writes goroutine and heap profiles to timestamped .pb.gz files
// in the current directory.
func DumpAllCmd(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		ts := time.Now().UTC().Format("20060102T150405Z")
		prefix := "ogle-profile-" + ts

		var cpuPath string

		if d > 0 {
			cpuPath = filepath.Join(".", prefix+"-cpu.pb.gz")

			f, createErr := os.Create(cpuPath)
			if createErr != nil {
				return ProfilesDumped{
					CPUProfilePath: "",
					GoroutinePath:  "",
					HeapPath:       "",
					Err:            fmt.Errorf("create CPU profile: %w", createErr),
				}
			}

			if startErr := pprof.StartCPUProfile(f); startErr != nil {
				_ = f.Close()
				_ = os.Remove(cpuPath)

				return ProfilesDumped{
					CPUProfilePath: "",
					GoroutinePath:  "",
					HeapPath:       "",
					Err:            fmt.Errorf("start CPU profile: %w", startErr),
				}
			}

			time.Sleep(d)

			pprof.StopCPUProfile()

			_ = f.Close()
		}

		goPath := filepath.Join(".", prefix+"-goroutine.pb.gz")
		if err := writeProfile("goroutine", goPath); err != nil {
			return ProfilesDumped{
				CPUProfilePath: cpuPath,
				GoroutinePath:  "",
				HeapPath:       "",
				Err:            fmt.Errorf("write goroutine profile: %w", err),
			}
		}

		heapPath := filepath.Join(".", prefix+"-heap.pb.gz")
		if err := writeProfile("heap", heapPath); err != nil {
			return ProfilesDumped{
				CPUProfilePath: cpuPath,
				GoroutinePath:  goPath,
				HeapPath:       "",
				Err:            fmt.Errorf("write heap profile: %w", err),
			}
		}

		return ProfilesDumped{
			CPUProfilePath: cpuPath,
			GoroutinePath:  goPath,
			HeapPath:       heapPath,
			Err:            nil,
		}
	}
}

func writeProfile(name, path string) error {
	f, createErr := os.Create(path)
	if createErr != nil {
		return fmt.Errorf("create %s: %w", name, createErr)
	}
	defer f.Close()

	if writeErr := pprof.Lookup(name).WriteTo(f, 0); writeErr != nil {
		return fmt.Errorf("write %s: %w", name, writeErr)
	}

	return nil
}
