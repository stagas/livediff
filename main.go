package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

const (
	reset    = "\x1b[0m"
	red      = "\x1b[31m"
	green    = "\x1b[32m"
	cyan     = "\x1b[36m"
	dim      = "\x1b[2m"
	interval = 200 * time.Millisecond
)

type entry struct {
	data    []byte
	modTime time.Time
	size    int64
}

type app struct {
	compact bool
	color   bool
	raw     bool
	cwd     string
	entries map[string]entry
}

func git(args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, errors.New(message)
	}
	return output, nil
}

func pathsInGit() ([]string, error) {
	output, err := git("ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	parts := bytes.Split(output, []byte{0})
	paths := make([]string, 0, len(parts))
	for _, path := range parts {
		if len(path) > 0 {
			paths = append(paths, string(path))
		}
	}
	return paths, nil
}

func readEntry(path string) (entry, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return entry{}, false, nil
		}
		return entry{}, false, err
	}
	if !info.Mode().IsRegular() {
		return entry{}, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return entry{}, false, nil
		}
		return entry{}, false, err
	}
	return entry{data: data, modTime: info.ModTime(), size: info.Size()}, true, nil
}

func isBinary(data []byte) bool {
	if len(data) > 8_000 {
		data = data[:8_000]
	}
	return bytes.IndexByte(data, 0) >= 0
}

func (a *app) paint(color, value string) string {
	if !a.color {
		return value
	}
	return color + value + reset
}

func (a *app) write(value string) {
	if a.raw {
		value = strings.ReplaceAll(value, "\n", "\r\n")
	}
	fmt.Fprint(os.Stdout, value)
}

func (a *app) printf(format string, values ...any) {
	a.write(fmt.Sprintf(format, values...))
}

func (a *app) println(value string) {
	a.write(value + "\n")
}

func printablePath(path string) string {
	for _, character := range path {
		if character < 0x20 || character == 0x7f {
			return strconv.Quote(path)
		}
	}
	return path
}

func (a *app) show(path string, before, after []byte, existedBefore, existsAfter bool) {
	path = printablePath(path)
	now := time.Now().Format("3:04:05 PM")
	if a.compact {
		a.printf("\n%s %s\n", a.paint(cyan, path), a.paint(dim, now))
	} else {
		a.printf("\n%s %s\n", a.paint(cyan, "diff --git a/"+path+" b/"+path), a.paint(dim, now))
		oldPath, newPath := "/dev/null", "/dev/null"
		if existedBefore {
			oldPath = "a/" + path
		}
		if existsAfter {
			newPath = "b/" + path
		}
		a.println(a.paint(red, "--- "+oldPath))
		a.println(a.paint(green, "+++ "+newPath))
	}

	if isBinary(before) || isBinary(after) {
		if a.compact {
			a.println("binary changed")
		} else {
			a.println("Binary file changed")
		}
		return
	}

	context := 3
	if a.compact {
		context = 0
	}
	for _, hunk := range makeHunks(diffLines(string(before), string(after)), context) {
		if !a.compact {
			header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.oldStart, hunk.oldCount, hunk.newStart, hunk.newCount)
			a.println(a.paint(cyan, header))
		}
		for _, line := range hunk.lines {
			switch line.kind {
			case add:
				a.println(a.paint(green, "+"+line.text))
			case remove:
				a.println(a.paint(red, "-"+line.text))
			case equal:
				if !a.compact {
					a.println(" " + line.text)
				}
			}
		}
	}
}

func (a *app) scan() error {
	paths, err := pathsInGit()
	if err != nil {
		return err
	}
	current := make(map[string]bool, len(paths))
	for _, path := range paths {
		current[path] = true
	}

	all := make(map[string]bool, len(a.entries)+len(current))
	for path := range a.entries {
		all[path] = true
	}
	for path := range current {
		all[path] = true
	}

	for path := range all {
		previous, hadPrevious := a.entries[path]
		if !current[path] {
			if hadPrevious {
				a.show(path, previous.data, nil, true, false)
			}
			delete(a.entries, path)
			continue
		}

		absolute := filepath.Join(a.cwd, filepath.FromSlash(path))
		info, statErr := os.Stat(absolute)
		if statErr != nil || !info.Mode().IsRegular() {
			if hadPrevious && (statErr == nil || errors.Is(statErr, os.ErrNotExist)) {
				a.show(path, previous.data, nil, true, false)
				delete(a.entries, path)
			}
			continue
		}
		if hadPrevious && previous.size == info.Size() && previous.modTime.Equal(info.ModTime()) {
			continue
		}

		next, ok, readErr := readEntry(absolute)
		if readErr != nil {
			return readErr
		}
		if !ok {
			continue
		}
		if !hadPrevious || !bytes.Equal(previous.data, next.data) {
			a.show(path, previous.data, next.data, hadPrevious, true)
		}
		a.entries[path] = next
	}
	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: livediff [options]\n\nOptions:\n")
		fmt.Fprintln(flag.CommandLine.Output(), "  -c, --compact  show only the filename, time, and changed lines")
		fmt.Fprintln(flag.CommandLine.Output(), "  -h, --help     show this help")
	}
	var compact bool
	flag.BoolVar(&compact, "compact", false, "")
	flag.BoolVar(&compact, "c", false, "")
	flag.Parse()

	if _, err := git("rev-parse", "--is-inside-work-tree"); err != nil {
		fmt.Fprintln(os.Stderr, "livediff: the current directory is not inside a Git work tree")
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "livediff:", err)
		os.Exit(1)
	}

	a := app{
		compact: compact,
		color:   term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == "",
		cwd:     cwd,
		entries: make(map[string]entry),
	}
	paths, err := pathsInGit()
	if err != nil {
		fmt.Fprintln(os.Stderr, "livediff:", err)
		os.Exit(1)
	}
	for _, path := range paths {
		if item, ok, readErr := readEntry(filepath.Join(cwd, filepath.FromSlash(path))); readErr != nil {
			fmt.Fprintln(os.Stderr, "livediff:", readErr)
		} else if ok {
			a.entries[path] = item
		}
	}

	a.println(a.paint(cyan, "livediff watching "+cwd))
	a.println(a.paint(dim, "Press c to clear; q or Ctrl-C to quit."))

	stop := make(chan os.Signal, 1)
	clearScreen := make(chan struct{}, 1)
	signal.Notify(stop, os.Interrupt, syscall.Signal(15))
	defer signal.Stop(stop)

	if term.IsTerminal(int(os.Stdin.Fd())) {
		if oldState, rawErr := term.MakeRaw(int(os.Stdin.Fd())); rawErr == nil {
			a.raw = true
			defer func() {
				a.raw = false
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
			}()
			go func() {
				buffer := make([]byte, 1)
				for {
					if _, readErr := os.Stdin.Read(buffer); readErr != nil {
						return
					}
					if buffer[0] == 'q' || buffer[0] == 'Q' || buffer[0] == 3 {
						stop <- os.Interrupt
						return
					}
					if buffer[0] == 'c' || buffer[0] == 'C' {
						select {
						case clearScreen <- struct{}{}:
						default:
						}
					}
				}
			}()
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			a.println(a.paint(dim, "\nStopped."))
			return
		case <-clearScreen:
			a.write("\x1b[2J\x1b[H")
		case <-ticker.C:
			if err := a.scan(); err != nil {
				fmt.Fprintln(os.Stderr, "livediff:", err)
			}
		}
	}
}
