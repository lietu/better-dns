package main

import (
	"bufio"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/getlantern/systray"
	"github.com/lietu/better-dns/icon"
	"github.com/lietu/better-dns/shared"
	"github.com/lietu/better-dns/stats"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	runtime.LockOSThread()
	start()
}

func start() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Caught panic: %s", err)
		}
	}()

	logFile := path.Join(shared.GetConfigDir(), "better-dns-tray.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to logger to file, using default stderr")
	}

	onExit := func() {
		log.Info("Exiting")
	}

	log.Info("Starting up")

	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s := <-sig
		log.Infof("Caught signal %s", s)
		systray.Quit()
	}()

	systray.Run(onReady, onExit)
}

func onReady() {
	runner := newBetterDnsRunner()
	runner.Start()

	systray.SetIcon(icon.Unknown)

	systray.SetTitle("Better DNS")
	systray.SetTooltip("Better DNS manager")

	// We can manipulate the systray in other goroutines
	itemStatus := systray.AddMenuItem("Starting...", "Current status")
	systray.AddSeparator()
	itemRequests := systray.AddMenuItem("", "Total DNS requests")
	itemSuccesses := systray.AddMenuItem("", "Total successfully resolved DNS requests")
	itemBlocked := systray.AddMenuItem("", "Total blocked DNS requests")
	itemCached := systray.AddMenuItem("", "Total cached DNS requests")
	itemErrors := systray.AddMenuItem("", "Total DNS requests that resulted in errors")
	systray.AddSeparator()
	menuToggleState := systray.AddMenuItem("", "Start/stop Better DNS")
	menuQuit := systray.AddMenuItem("Quit", "Close Better DNS manager")

	running := false

	for {
		select {
		case state := <-runner.stateCn:
			running = state.Running
			status := "Not running"
			if running {
				status = "Running"
				systray.SetIcon(icon.On)
				menuToggleState.SetTitle("Pause Better DNS")
			} else {
				systray.SetIcon(icon.Off)
				menuToggleState.SetTitle("Resume Better DNS")
			}

			if state.Stats == nil {
				continue
			}

			s := state.Stats
			total := s.Errors + s.Cached + s.Blocked + s.Successes

			blockPct := stats.RequestPct(s.Blocked, total)
			cachePct := stats.RequestPct(s.Cached, total)
			errorPct := stats.RequestPct(s.Errors, total)

			itemStatus.SetTitle(status)
			itemRequests.SetTitle(fmt.Sprintf("%s requests", humanize.Comma(int64(total))))
			itemSuccesses.SetTitle(fmt.Sprintf("%s successful (avg %s)", humanize.Comma(int64(s.Successes)), s.Rtt))
			itemBlocked.SetTitle(fmt.Sprintf("%s blocked (%s)", humanize.Comma(int64(s.Blocked)), blockPct))
			itemCached.SetTitle(fmt.Sprintf("%s cached (%s)", humanize.Comma(int64(s.Cached)), cachePct))
			itemErrors.SetTitle(fmt.Sprintf("%s errors (%s)", humanize.Comma(int64(s.Errors)), errorPct))

		case <-menuToggleState.ClickedCh:
			if running {
				runner.Stop(false)
			} else {
				runner.Start()
			}

		case <-menuQuit.ClickedCh:
			runner.Stop(true)
			systray.Quit()
			return
		}
	}
}

type runnerState struct {
	Running bool
	Stats   *stats.Stats
}

type betterDnsRunner struct {
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stateCn      chan *runnerState
	state        *runnerState
	exitStdoutCn chan bool
	exitStderrCn chan bool
}

func newBetterDnsRunner() *betterDnsRunner {
	r := betterDnsRunner{}
	r.stateCn = make(chan *runnerState)
	r.exitStdoutCn = make(chan bool, 2)
	r.exitStderrCn = make(chan bool, 2)
	r.state = &runnerState{
		Running: false,
		Stats:   nil,
	}
	return &r
}

func (r *betterDnsRunner) Start() {
	if r.cmd != nil && r.cmd.ProcessState != nil && r.cmd.ProcessState.Exited() == false {
		log.Info("better-dns already running")
		// Already running
		return
	}

	log.Info("Starting better-dns")

	r.cmd = exec.Command("better-dns", "-tray")
	reader, err := r.cmd.StdoutPipe()
	if err != nil {
		log.Errorf("Failed to get stdout pipe for better-dns: %s", err)
	}

	stderr, err := r.cmd.StderrPipe()
	if err != nil {
		log.Errorf("Failed to get stderr pipe for better-dns: %s", err)
	}

	stdin, err := r.cmd.StdinPipe()
	if err != nil {
		log.Errorf("Failed to get stdin pipe for better-dns: %s", err)
	}
	r.stdin = stdin

	r.cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	go func(reader io.ReadCloser) {
		scanner := bufio.NewScanner(reader)
		for {
			select {
			case <-r.exitStdoutCn:
				return
			default:
				if scanner.Scan() {
					line := scanner.Text()

					parts := strings.Split(line, ",")
					if len(parts) < 5 {
						// Unknown line
						continue
					}

					state := r.state
					state.Running = true
					state.Stats = &stats.Stats{}

					for _, p := range parts {
						if strings.HasPrefix(p, "S:") {
							v, err := strconv.ParseInt(strings.TrimPrefix(p, "S:"), 10, 64)
							if err == nil {
								state.Stats.Successes = uint64(v)
							}
						} else if strings.HasPrefix(p, "B:") {
							v, err := strconv.ParseInt(strings.TrimPrefix(p, "B:"), 10, 64)
							if err == nil {
								state.Stats.Blocked = uint64(v)
							}
						} else if strings.HasPrefix(p, "C:") {
							v, err := strconv.ParseInt(strings.TrimPrefix(p, "C:"), 10, 64)
							if err == nil {
								state.Stats.Cached = uint64(v)
							}
						} else if strings.HasPrefix(p, "E:") {
							v, err := strconv.ParseInt(strings.TrimPrefix(p, "E:"), 10, 64)
							if err == nil {
								state.Stats.Errors = uint64(v)
							}
						} else if strings.HasPrefix(p, "R:") {
							v, err := strconv.ParseInt(strings.TrimPrefix(p, "R:"), 10, 64)
							if err == nil {
								state.Stats.Rtt = time.Millisecond * time.Duration(v)
							}
						}
					}

					r.state = state
					go r.SendState()
				} else {
					// Pipe closed
					r.state.Running = false
					go r.SendState()
					return
				}
			}
		}
	}(reader)

	go func(reader io.ReadCloser) {
		scanner := bufio.NewScanner(reader)
		for {
			select {
			case <-r.exitStderrCn:
				return
			default:
				if scanner.Scan() {
					line := scanner.Text()
					log.Printf("Got stderr: '%s'", line)
				} else {
					// Pipe closed
					r.state.Running = false
					go r.SendState()
					return
				}
			}
		}
	}(stderr)

	if err := r.cmd.Start(); err != nil {
		log.Errorf("Error starting better-dns: %s", err)
	}
}

func (r *betterDnsRunner) Stop(exit bool) {
	if r.cmd != nil {
		log.Info("Stopping better-dns")

		if r.stdin != nil {
			if _, err := r.stdin.Write([]byte("exit\n")); err != nil {
				log.Errorf("Failed to write exit command to better-dns: %s", err)
			}
		} else if err := r.cmd.Process.Signal(syscall.SIGKILL); err != nil {
			log.Errorf("Error signaling better-dns: %s", err)
		}

		if exit {
			log.Info("Sending exit signals")
			r.exitStdoutCn <- true
			r.exitStderrCn <- true
		}

		log.Info("Waiting for better-dns to exit")
		if err := r.cmd.Wait(); err != nil {
			log.Errorf("Error waiting for better-dns to stop: %s", err)
		}

		log.Info("Sending update after close")
		r.cmd = nil
		r.state.Running = false
		go r.SendState()
	}
}

func (r *betterDnsRunner) SendState() {
	r.stateCn <- r.state
}
