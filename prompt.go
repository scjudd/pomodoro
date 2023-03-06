package main

import (
	"bufio"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

type prompt struct {
	prompt string
	cols   int
	bottom int
	buf    []rune
	start  int
	cur    int
}

func newPrompt(promptString string, rows, cols int) *prompt {
	return &prompt{prompt: promptString, cols: cols, bottom: rows}
}

func newPromptWithLine(promptString, line string, rows, cols int) *prompt {
	p := newPrompt(promptString, rows, cols)
	p.buf = []rune(line)
	p.cur = len(p.buf)
	p.start = len(p.buf) - p.bufferWidth()
	if p.start < 0 {
		p.start = 0
	}
	return p
}

func (p *prompt) bufferWidth() int {
	return p.cols - utf8.RuneCountInString(p.prompt) - 1
}

func (p *prompt) bufferSlice() []rune {
	for p.cur < p.start {
		p.start--
	}
	for p.cur > p.start+p.bufferWidth() {
		p.start++
	}
	end := p.start + p.bufferWidth()
	if end > len(p.buf) {
		end = len(p.buf)
	}
	return p.buf[p.start:end]
}

func (p *prompt) resize(rows, cols int) {
	p.bottom = rows
	p.cols = cols
	p.redraw()
}

func (p *prompt) redraw() {
	// If there aren't enough columns to display the prompt, bail.
	if utf8.RuneCountInString(p.prompt) > p.cols {
		return
	}

	var sb strings.Builder

	sb.WriteString(escCursorHide)
	sb.WriteString(escCursorMove(p.bottom, 0))
	sb.WriteString(escClearLine)

	sb.WriteString(p.prompt)
	sb.WriteString(string(p.bufferSlice()))

	cur := p.cur - p.start + utf8.RuneCountInString(p.prompt) + 1
	sb.WriteString(escCursorMove(p.bottom, cur))
	sb.WriteString(escCursorShow)

	_, err := io.WriteString(os.Stdout, sb.String())
	if err != nil {
		panic(err)
	}
}

func (p *prompt) insert(r rune) {
	if len(p.buf) == p.cur {
		p.buf = append(p.buf, r)
	} else {
		p.buf = append(p.buf[:p.cur+1], p.buf[p.cur:]...)
		p.buf[p.cur] = r
	}
	p.cur++
}

func (p *prompt) moveLeft() {
	if p.cur > 0 {
		p.cur--
	}
}

func (p *prompt) moveRight() {
	if p.cur < len(p.buf) {
		p.cur++
	}
}

func (p *prompt) moveHome() {
	if p.cur > 0 {
		p.cur = 0
	}
}

func (p *prompt) moveEnd() {
	if p.cur < len(p.buf) {
		p.cur = len(p.buf)
	}
}

func (p *prompt) backspace() {
	if p.cur > 0 && len(p.buf) > 0 {
		p.buf = append(p.buf[:p.cur-1], p.buf[p.cur:]...)
		p.cur--
	}
}

func (p *prompt) deletePrevWord() {
	oldCur := p.cur
	for p.cur > 0 && p.buf[p.cur-1] == ' ' {
		p.cur--
	}
	for p.cur > 0 && p.buf[p.cur-1] != ' ' {
		p.cur--
	}
	p.buf = append(p.buf[:p.cur], p.buf[oldCur:]...)
}

const (
	backspace = '\x7f'
	ctrlA     = '\x01'
	ctrlE     = '\x05'
	ctrlW     = '\x17'
	enter     = '\x0a'
	esc       = '\x1b'
)

// Control Sequence Introducer (CSI)
const (
	csi      = '['
	csiLeft  = 'D'
	csiRight = 'C'
)

func (p *prompt) read() string {
	in := bufio.NewReader(os.Stdin)
main:
	for {
		p.redraw()

		r, _, err := in.ReadRune()
		if err != nil {
			panic(err)
		}

		switch r {
		case backspace:
			p.backspace()
		case ctrlA:
			p.moveHome()
		case ctrlE:
			p.moveEnd()
		case ctrlW:
			p.deletePrevWord()
		case enter:
			break main
		case esc:
			r, _, err = in.ReadRune()
			if err != nil {
				panic(err)
			}
			if r != csi {
				panic("non-CSI escape sequence")
			}

			r, _, err = in.ReadRune()
			if err != nil {
				panic(err)
			}
			switch r {
			case csiLeft:
				p.moveLeft()
			case csiRight:
				p.moveRight()
			default:
				break
			}
		default:
			p.insert(r)
		}
	}

	return strings.TrimSpace(string(p.buf))
}
