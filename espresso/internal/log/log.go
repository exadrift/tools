package log

import (
	"github.com/exadrift/go/ansi/style"
	"github.com/exadrift/go/statusline"
)

type Condition string

const (
	ConditionPass Condition = "✅"
	ConditionWarn Condition = "🟠"
	ConditionFail Condition = "❌"
)

type Log struct {
	statusline *statusline.StatusLine
}

func New() *Log {
	return &Log{
		statusline: statusline.NewStatusLine(statusline.WithSpinnerStyling(style.Yellow.Fg().Ansi)),
	}
}

func (l *Log) Status(t style.Text) {
	l.statusline.Status(t.Render()[0])
}

func (l *Log) Emit(t style.Text) {
	text := t.Render()
	// always reset style
	l.statusline.Emit(text[0] + "\x1b[0m")
}

func (l *Log) EmitRaw(text string) {
	l.statusline.Emit(text)
}

func (l *Log) EmitCondition(condition Condition, message style.Text) {
	l.Emit(style.T(string(condition), " ").Extend(message))
}

func (l *Log) Prompt(text style.Text) string {
	return l.statusline.Prompt(text.Render()[0], true)
}

func (l *Log) Stop() {
	l.statusline.Stop()
}
