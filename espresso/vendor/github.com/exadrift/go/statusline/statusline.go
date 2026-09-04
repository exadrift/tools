package statusline

import (
	"bufio"
	"fmt"
	"os"
	"sync"
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
)

type StatusMessage struct {
	Message  string
	Type     MessageType
	Callback func(resp string, err error)
	Secret   bool
}

type StatusLine struct {
	wg          sync.WaitGroup
	options     StatusLineOptions
	messageChan chan StatusMessage
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

func (s *StatusLine) Stop() {
	close(s.messageChan)
	s.wg.Wait()
}

func (s *StatusLine) displayStatusMessage(roller string, message string) {
	fmt.Printf("\r%s%s\x1b[0m %s\x1b[0m\x1b[K", s.options.SpinnerStyling, roller, message)
}

func (s *StatusLine) emitMessage(message string) {
	fmt.Printf("\r\x1b[K%s\n", message)
}

func (s *StatusLine) promptMessage(message string, secret bool) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("\r\x1b[K\r%s ", message)
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

func (s *StatusLine) runLoop() {
	defer s.wg.Done()
	defer fmt.Printf("\x1b[?25h")
	var statusMessage string
	var rollerIndex int

	ticker := time.NewTicker(TickerTime)

	fmt.Printf("\x1b[?25l")

	for {
		select {
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

// Status displays a persistent status message with a roller, indicating that the status is ongoing.
// Status will not advance the current line when updated and thus status can be updated with new
// messages on the same line as the status changees
func (s *StatusLine) Status(message string) {
	s.messageChan <- StatusMessage{message, MessageTypeStatus, nil, false}
}

// Emit emits a new message in the style of a running log.  It will clear the current line and
// remove any status message which may be present.  Typically the use case is to emit a message
// once it enters the log record, in order to conclude the current ongoing operation
func (s *StatusLine) Emit(message string) {
	s.messageChan <- StatusMessage{message, MessageTypeEmit, nil, false}
}

// Prompt prompts the user for input and returns it
func (s *StatusLine) Prompt(message string, secret bool) string {
	var promptResp string
	wg := sync.WaitGroup{}
	wg.Add(1)
	s.messageChan <- StatusMessage{message, MessageTypePrompt, func(resp string, err error) {
		defer wg.Done()
		promptResp = resp
	}, secret}
	wg.Wait()

	return promptResp
}
