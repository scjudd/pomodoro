package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const pomodoroDuration = 15 * time.Minute

var tasks = []task{}
var debugEnabled = false

type mode int

const (
	modeNormal mode = iota
	modeEditCompleted
	modeDeleteConfirm
	modeReorder
)

type userInterface struct {
	mode           mode
	promptInstance *prompt
	statusLine     string
	selected       int
	active         int
	running        bool
	timeRemaining  time.Duration
	rows           int
	cols           int
	cursorRow      int
	cursorCol      int
}

var ui = userInterface{
	timeRemaining: pomodoroDuration,
}

func (ui *userInterface) changeMode(m mode) {
	ui.mode = m
	switch ui.mode {
	case modeNormal:
		ui.statusLine = ""
	case modeEditCompleted:
		ui.statusLine = "Edit completed pomodoros with +/-"
	case modeDeleteConfirm:
		ui.statusLine = "Delete selected task?"
	case modeReorder:
		ui.statusLine = "Moving selected task..."
	}
}

func (ui *userInterface) decrementTimer() {
	ui.timeRemaining -= time.Second
	if ui.timeRemaining == 0 {
		ui.running = false
		tasks[ui.active].completed++
		if tasks[ui.active].completed > tasks[ui.active].pomodoros {
			tasks[ui.active].pomodoros = tasks[ui.active].completed
		}
		ui.resetTimer()
	}
}

func (ui *userInterface) resetTimer() {
	ui.running = false
	ui.timeRemaining = pomodoroDuration
}

func (ui *userInterface) moveCursor(row, col int) {
	fmt.Print(escCursorMove(row, col))
	ui.cursorRow, ui.cursorCol = row, col
}

func (ui *userInterface) prompt(prompt, current string) (result string) {
	ui.promptInstance = newPromptWithLine(prompt, current, ui.rows, ui.cols)
	result = ui.promptInstance.read()
	ui.promptInstance = nil
	return
}

func (ui *userInterface) resize() {
	ui.rows, ui.cols = getWindowSize()
	ui.redraw()
	if ui.promptInstance != nil {
		ui.promptInstance.resize(ui.rows, ui.cols)
	}
}

func (ui *userInterface) redraw() {
	fmt.Print(escClearDisplay, escCursorHide)

	if len(tasks) == 0 {
		msg := "No tasks added. Press 'a' to create a new task."
		ui.moveCursor(ui.rows/2, (ui.cols-len(msg))/2)
		if len(msg) > ui.cols {
			fmt.Print(msg[:ui.cols-3], "...")
		} else {
			fmt.Print(msg)
		}
		return
	}

	timerTop := (ui.rows - len(tasks)) / 2
	if ui.running {
		fmt.Print(escSgrReverseVideo)
	}
	msg := fmt.Sprintf("Time remaining: %s", ui.timeRemaining.String())
	ui.moveCursor(timerTop, (ui.cols-len(msg))/2)
	if len(msg) > ui.cols {
		fmt.Print(msg[:ui.cols-3], "...", escSgrReset)
	} else {
		fmt.Print(msg, escSgrReset)
	}

	maxPomodoroString, maxDescription := 0, 0
	for _, task := range tasks {
		pomoStringLen := utf8.RuneCountInString(pomodoroString(task))
		if pomoStringLen > maxPomodoroString {
			maxPomodoroString = pomoStringLen
		}
		if len(task.description) > maxDescription {
			maxDescription = len(task.description)
		}
	}

	const selectionMarker = "› "
	var selectionMarkerLen = utf8.RuneCountInString(selectionMarker)

	lineLength := selectionMarkerLen + maxPomodoroString + maxDescription
	if lineLength > ui.cols {
		maxDescription = ui.cols - selectionMarkerLen - maxPomodoroString
		lineLength = ui.cols
	}

	listTop := timerTop + 2
	listLeft := (ui.cols - lineLength) / 2
	ui.moveCursor(listTop, listLeft)

	for index, task := range tasks {
		ui.moveCursor(listTop+index, listLeft)
		if index == ui.selected {
			fmt.Print(selectionMarker)
		}

		ui.moveCursor(ui.cursorRow, ui.cursorCol+selectionMarkerLen)
		fmt.Print(escSgrForegroundRed, pomodoroString(task), escSgrReset)

		ui.moveCursor(ui.cursorRow, ui.cursorCol+maxPomodoroString)
		if index == ui.active {
			fmt.Print(escSgrReverseVideo)
		}
		if len(task.description) > maxDescription {
			fmt.Print(task.description[:maxDescription-3], "...", escSgrReset)
		} else {
			fmt.Print(task.description, escSgrReset)
		}
	}

	if ui.promptInstance != nil {
		ui.promptInstance.redraw()
	} else {
		ui.moveCursor(ui.rows, 0)
		fmt.Print(escClearLine, ui.statusLine)
	}
}

func pomodoroString(task task) string {
	var sb strings.Builder
	if task.completed > 0 {
		sb.WriteString("⬢ ")
		sb.WriteString(strconv.Itoa(task.completed))
		sb.WriteString(" ")
	}
	if task.pomodoros-task.completed > 0 {
		sb.WriteString("⬡ ")
		sb.WriteString(strconv.Itoa(task.pomodoros - task.completed))
		sb.WriteString(" ")
	}
	return sb.String()
}

func actionAddTask() {
	description := ui.prompt("Add > ", "")
	if len(description) > 0 {
		tasks = append(tasks, task{pomodoros: 1, description: description})
	}
}

func actionChangeCompletedPomodoros(relative int) {
	tasks[ui.selected].completed += relative
	if tasks[ui.selected].completed < 0 {
		tasks[ui.selected].completed = 0
	} else if tasks[ui.selected].completed > tasks[ui.selected].pomodoros {
		tasks[ui.selected].completed = tasks[ui.selected].pomodoros
	}
}

func actionChangeMode(m mode) {
	ui.changeMode(m)
}

func actionChangePomodoros(relative int) {
	tasks[ui.selected].pomodoros += relative
	if tasks[ui.selected].pomodoros < 1 {
		tasks[ui.selected].pomodoros = 1
	}
	if tasks[ui.selected].completed > tasks[ui.selected].pomodoros {
		tasks[ui.selected].pomodoros = tasks[ui.selected].completed
	}
}

func actionChangeSelection(relative int) {
	ui.selected += relative
	if ui.selected < 0 {
		ui.selected = 0
	} else if ui.selected > len(tasks)-1 {
		ui.selected = len(tasks) - 1
	}
}

func actionStartStop() {
	if ui.running {
		ui.running = false
	} else {
		ui.running = true
		ui.active = ui.selected
	}
}

func actionDeleteTask() {
	tasks = append(tasks[:ui.selected], tasks[ui.selected+1:]...)
	if ui.selected == ui.active {
		ui.resetTimer()
		ui.active--
		if ui.active < 0 {
			ui.active = 0
		}
	}
	ui.selected--
	if ui.selected < 0 {
		ui.selected = 0
	}
}

func actionEditTaskDescription() {
	newDescription := ui.prompt("Edit > ", tasks[ui.selected].description)
	if len(newDescription) > 0 {
		tasks[ui.selected].description = newDescription
	}
}

func actionMoveTask(relative int) {
	thisIndex, otherIndex := ui.selected, ui.selected+relative
	if otherIndex < 0 || otherIndex > len(tasks)-1 {
		return
	}
	tasks[otherIndex], tasks[thisIndex] = tasks[thisIndex], tasks[otherIndex]
	ui.selected = otherIndex
	if ui.active == thisIndex {
		ui.active = otherIndex
	} else if ui.active == otherIndex {
		ui.active = thisIndex
	}
}

func actionResetTimer() {
	ui.resetTimer()
}

func handleInput(r rune) {
	switch ui.mode {
	case modeNormal:
		handleInputNormal(r)
	case modeEditCompleted:
		handleInputEditCompleted(r)
	case modeDeleteConfirm:
		handleInputDeleteConfirm(r)
	case modeReorder:
		handleInputReorder(r)
	}
}

func handleInputNormal(r rune) {
	if len(tasks) == 0 && r != 'a' {
		// No other action is valid if we don't have any tasks yet.
		return
	}

	switch r {
	case 'a':
		actionAddTask()
	case 'd':
		actionChangeMode(modeDeleteConfirm)
	case 'e':
		actionEditTaskDescription()
	case 'j':
		actionChangeSelection(1)
	case 'k':
		actionChangeSelection(-1)
	case 'm':
		actionChangeMode(modeReorder)
	case 's':
		actionStartStop()
	case 'r':
		actionResetTimer()
	case 'E':
		actionChangeMode(modeEditCompleted)
	case '-':
		actionChangePomodoros(-1)
	case '+':
		actionChangePomodoros(1)
	}
}

func handleInputEditCompleted(r rune) {
	switch r {
	case '-':
		actionChangeCompletedPomodoros(-1)
	case '+':
		actionChangeCompletedPomodoros(1)
	case '\n':
		actionChangeMode(modeNormal)
	}
}

func handleInputDeleteConfirm(r rune) {
	if r == 'd' || r == 'y' {
		actionDeleteTask()
	}
	actionChangeMode(modeNormal)
}

func handleInputReorder(r rune) {
	switch r {
	case 'j':
		actionMoveTask(1)
	case 'k':
		actionMoveTask(-1)
	case '\n':
		actionChangeMode(modeNormal)
	}
}

func handleSignal(sig os.Signal, handler func()) {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, sig)
		for range c {
			handler()
		}
	}()
}

func timerLoop() {
	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C
		if ui.running {
			ui.decrementTimer()
			ui.redraw()
		}
	}
}

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "debug" {
		debugEnabled = true
	}

	handleSignal(syscall.SIGWINCH, ui.resize)

	handleSignal(os.Interrupt, func() {
		fmt.Print(escXtermAlternativeScreenDisable, escCursorShow)
		restoreTerminalMode()
		os.Exit(0)
	})

	fmt.Print(escXtermAlternativeScreenEnable, escCursorHide)
	enterRawTerminalMode()
	ui.resize()

	go timerLoop()

	in := bufio.NewReader(os.Stdin)
	for {
		r, _, err := in.ReadRune()
		if err != nil {
			panic(err)
		}
		handleInput(r)
		ui.redraw()
	}
}

func debug(msg string) {
	if !debugEnabled {
		return
	}
	io.WriteString(os.Stderr, msg)
	io.WriteString(os.Stderr, "\n")
}
