package statusline

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

const (
	TickerTime = 100 * time.Millisecond
)

type StatusLineOptions struct {
	SpinnerSet     []string
	SpinnerStyling string
}

type MessageType int

const (
	MessageTypeStatus MessageType = iota
	MessageTypeEmit
	MessageTypePrompt
	MessageTypeConfirm
)

type StatusMessage struct {
	Message      string
	Type         MessageType
	Callback     func(resp string, err error)
	Secret       bool
	DefaultValue bool
}

type StatusLine struct {
	wg          sync.WaitGroup
	options     StatusLineOptions
	messageChan chan StatusMessage
	startTime   time.Time
	holdTimer   bool
	lock        sync.Mutex
	width       int
}

var defaultSpinner = []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"}

type Option func(opt *StatusLineOptions)

func WithSpinner(spinnerSet []string) Option {
	return func(opt *StatusLineOptions) {
		opt.SpinnerSet = spinnerSet
	}
}

func WithSpinnerStyling(style string) Option {
	return func(opt *StatusLineOptions) {
		opt.SpinnerStyling = style
	}
}

func NewStatusLine(options ...Option) *StatusLine {
	s := &StatusLine{
		messageChan: make(chan StatusMessage, 100),
	}
	s.options.SpinnerSet = defaultSpinner

	for _, opt := range options {
		opt(&s.options)
	}

	s.wg.Add(1)
	go s.runLoop()

	return s
}

func (s *StatusLine) GetWidth() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.width
}

func (s *StatusLine) SetWidth(width int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.width = width
}

func (s *StatusLine) Stop() {
	close(s.messageChan)
	s.wg.Wait()
}

func (s *StatusLine) HoldTimer() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.holdTimer = true
}

func (s *StatusLine) ReleaseTimer() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.holdTimer = false
}

func (s *StatusLine) displayStatusMessage(roller string, message string) {
	var since string
	s.lock.Lock()
	st := time.Since(s.startTime).Round(time.Millisecond * 10)
	if st < time.Second {
		since = fmt.Sprintf("%0.1fs", st.Seconds())
	} else {
		since = st.String()
	}
	width := s.width

	s.lock.Unlock()

	base := fmt.Sprintf("%s %s ", roller, since)
	baseLen := len(base)
	if len(message)+baseLen > width {
		maxMsg := width - baseLen
		if maxMsg > 0 {
			message = message[:maxMsg]
		}
	}

	fmt.Printf("\r%s%s\x1b[0m \x1b[0;38;2;87;147;244;49m%s\x1b[0m %s\x1b[0m\x1b[K", s.options.SpinnerStyling, roller, since, message)
}

func (s *StatusLine) emitMessage(message string) {
	fmt.Printf("\r\x1b[K\x1b[0;38;2;25;105;179;49m%s\x1b[0m %s\n", time.Now().Format(time.TimeOnly), message)
}

func (s *StatusLine) promptMessage(message string, secret bool) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("\r\x1b[K\r%s \x1b[?25h", message)
	defer fmt.Printf("\x1b[?25l")
	var line string
	if !secret {
		if scanner.Scan() {
			line = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("unable to read string from stdin")
		}

		return line, nil

	}

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("can't prompt for input, not a TTY")
	}

	bp, err := term.ReadPassword(fd)
	if err != nil {
		return "", err
	}
	fmt.Println()
	return string(bp), nil
}

func (s *StatusLine) confirmMessage(message string, defaultValue bool) (string, error) {
	defer fmt.Printf("\x1b[?25l")
	var line string
	for {
		line = ""
		scanner := bufio.NewScanner(os.Stdin)
		var confStr string
		if defaultValue {
			confStr = "(Y/n)"
		} else {
			confStr = "(y/N)"
		}
		fmt.Printf("\r\x1b[K\r%s %s: \x1b[?25h", message, confStr)

		if scanner.Scan() {
			line = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("unable to read string from stdin")
		}

		lowerLine := strings.ToLower(line)
		if lowerLine == "" {
			if defaultValue {
				return "y", nil
			}
			return "n", nil
		}

		if lowerLine == "y" || lowerLine == "yes" {
			return "y", nil
		}

		if lowerLine == "n" || lowerLine == "no" {
			return "n", nil
		}
	}
}

func (s *StatusLine) runLoop() {
	defer s.wg.Done()
	defer fmt.Printf("\x1b[?25h")
	var statusMessage string
	var rollerIndex int

	ticker := time.NewTicker(TickerTime)

	fmt.Printf("\x1b[?25l")

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		log.Fatalf("unable to interrogate terminal for size: %v", err)
	}
	s.SetWidth(width)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)

	for {
		select {
		case <-sigChan:
			width, _, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				log.Fatalf("unable to interrogate terminal for size: %v", err)
			}
			s.SetWidth(width)
		case m, ok := <-s.messageChan:
			// check if the channel has been closed
			if !ok {
				ticker.Stop()
				return
			}
			switch m.Type {
			case MessageTypeEmit:
				statusMessage = ""
				s.emitMessage(m.Message)
			case MessageTypeStatus:
				statusMessage = m.Message
				s.displayStatusMessage(s.options.SpinnerSet[rollerIndex], statusMessage)
			case MessageTypePrompt:
				ticker.Stop()
				m.Callback(s.promptMessage(m.Message, m.Secret))
				ticker.Reset(TickerTime)
			case MessageTypeConfirm:
				ticker.Stop()
				m.Callback(s.confirmMessage(m.Message, m.DefaultValue))
				ticker.Reset(TickerTime)
			}
		case <-ticker.C:
			if statusMessage != "" {
				rollerIndex++
				if rollerIndex >= len(s.options.SpinnerSet) {
					rollerIndex = 0
				}
				s.displayStatusMessage(s.options.SpinnerSet[rollerIndex], statusMessage)
			}
		}
	}
}

func prepareMessage(message string, args ...any) string {
	m := message
	if len(args) > 0 {
		m = fmt.Sprintf(message, args...)
	}

	return m
}

// Status displays a persistent status message with a roller, indicating that the status is ongoing.
// Status will not advance the current line when updated and thus status can be updated with new
// messages on the same line as the status changees
func (s *StatusLine) Status(message string, args ...any) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.holdTimer || s.startTime.IsZero() {
		s.startTime = time.Now()
	}

	s.messageChan <- StatusMessage{prepareMessage(message, args...), MessageTypeStatus, nil, false, false}
}

// Emit emits a new message in the style of a running log.  It will clear the current line and
// remove any status message which may be present.  Typically the use case is to emit a message
// once it enters the log record, in order to conclude the current ongoing operation
func (s *StatusLine) Emit(message string, args ...any) {
	s.messageChan <- StatusMessage{prepareMessage(message, args...), MessageTypeEmit, nil, false, false}
}

// Prompt prompts the user for input and returns it
func (s *StatusLine) Prompt(secret bool, message string, args ...any) string {
	var promptResp string
	wg := sync.WaitGroup{}
	wg.Add(1)
	s.messageChan <- StatusMessage{prepareMessage(message, args...), MessageTypePrompt, func(resp string, err error) {
		defer wg.Done()
		promptResp = resp
	}, secret, false}
	wg.Wait()

	return promptResp
}

// Confirm confirms a value
func (s *StatusLine) Confirm(defaultValue bool, message string, args ...any) bool {
	var promptResp bool
	wg := sync.WaitGroup{}
	wg.Add(1)
	s.messageChan <- StatusMessage{prepareMessage(message, args...), MessageTypeConfirm, func(resp string, err error) {
		defer wg.Done()
		if strings.ToLower(resp) == "y" {
			promptResp = true
		} else {
			promptResp = false
		}
	}, false, defaultValue}
	wg.Wait()

	return promptResp
}
