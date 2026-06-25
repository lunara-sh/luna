package ui

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// ANSI escape codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
)

var colorEnabled = true

func init() {
	if runtime.GOOS == "windows" {
		colorEnabled = os.Getenv("TERM") != ""
	}
}

func color(c, s string) string {
	if !colorEnabled {
		return s
	}
	return c + s + Reset
}

func Green_(s string) string  { return color(Green, s) }
func Red_(s string) string    { return color(Red, s) }
func Yellow_(s string) string { return color(Yellow, s) }
func Cyan_(s string) string   { return color(Cyan, s) }
func Bold_(s string) string   { return color(Bold, s) }
func Dim_(s string) string    { return color(Dim, s) }

func PrintBanner() {
	banner := `
  ██╗     ██╗   ██╗███╗   ██╗ █████╗
  ██║     ██║   ██║████╗  ██║██╔══██╗
  ██║     ██║   ██║██╔██╗ ██║███████║
  ██║     ██║   ██║██║╚██╗██║██╔══██║
  ███████╗╚██████╔╝██║ ╚████║██║  ██║
  ╚══════╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝`
	fmt.Println(Cyan_(banner))
	fmt.Println(Dim_("  Local development environment manager\n"))
}

func PrintHeader(text string) {
	fmt.Println()
	fmt.Println(Bold_(text))
	fmt.Println(strings.Repeat("─", 50))
}

func PrintFooter() {
	fmt.Println(Dim_("  Config stored at: ~/.luna/luna.json"))
	fmt.Println()
}

func PrintStatus(name, display string, running bool, pid, port int, portOpen bool) {
	dot := DotStopped()
	stateStr := Red_("stopped")
	if running || portOpen {
		dot = DotRunning()
		stateStr = Green_("running")
	}

	pidStr := Dim_("—")
	if pid > 0 && (running || portOpen) {
		pidStr = Dim_(fmt.Sprintf("pid %-6d", pid))
	}

	portStr := Dim_(fmt.Sprintf(":%d", port))
	if portOpen {
		portStr = Cyan_(fmt.Sprintf(":%d", port))
	}

	fmt.Printf(
		"  %s  %-22s %s  %s  %s\n",
		dot,
		Bold_(display),
		stateStr,
		pidStr,
		portStr,
	)
}

func PrintSuccess(msg string) {
	fmt.Printf("  %s %s\n", Green_("✓"), msg)
}

func PrintError(msg string) {
	fmt.Printf("  %s %s\n", Red_("✗"), msg)
}

func PrintInfo(msg string) {
	fmt.Printf("  %s %s\n", Cyan_("→"), msg)
}

func PrintWarn(msg string) {
	fmt.Printf("  %s %s\n", Yellow_("!"), msg)
}

type Spinner struct {
	frames []string
	msg    string
	done   chan struct{}
}

func NewSpinner(msg string) *Spinner {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	if runtime.GOOS == "windows" {
		frames = []string{"|", "/", "-", "\\"}
	}
	return &Spinner{frames: frames, msg: msg, done: make(chan struct{})}
}

func (s *Spinner) Start() {
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				fmt.Printf("\r%s\r", strings.Repeat(" ", len(s.msg)+5))
				return
			default:
				fmt.Printf("\r  %s %s", Cyan_(s.frames[i%len(s.frames)]), s.msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop(success bool, msg string) {
	close(s.done)
	time.Sleep(100 * time.Millisecond)
	if success {
		PrintSuccess(msg)
	} else {
		PrintError(msg)
	}
}

func DotRunning() string { return Green_("●") }
func DotStopped() string { return Red_("●") }
func DotUnknown() string { return Yellow_("●") }
