package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const dataFileName = "words.json"
const panelWidth = 54

const (
	colorReset  = "\033[0m"
	colorMuted  = "\033[38;5;245m"
	colorCyan   = "\033[38;5;81m"
	colorGreen  = "\033[38;5;114m"
	colorYellow = "\033[38;5;221m"
	colorRed    = "\033[38;5;203m"
	colorBold   = "\033[1m"
)

type Card struct {
	English   string    `json:"english"`
	Japanese  string    `json:"japanese"`
	CreatedAt time.Time `json:"created_at"`
}

type menuItem struct {
	key   string
	label string
	color string
}

func main() {
	rand.Seed(time.Now().UnixNano())

	cards, err := loadCards()
	if err != nil {
		exitWithError(err)
	}

	if len(os.Args) > 1 {
		if err := runCommand(os.Args[1], cards); err != nil {
			exitWithError(err)
		}
		return
	}

	if err := runMenu(cards); err != nil {
		exitWithError(err)
	}
}

func runCommand(command string, cards []Card) error {
	switch command {
	case "add":
		return addCardInteractive(cards)
	case "list":
		printCards(cards)
		return nil
	case "flip":
		return runFlipCards(cards)
	case "help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func runMenu(cards []Card) error {
	reader := bufio.NewReader(os.Stdin)
	items := []menuItem{
		{key: "1", label: "単語を追加", color: colorGreen},
		{key: "2", label: "単語一覧を見る", color: colorCyan},
		{key: "3", label: "フリップカードで学習", color: colorYellow},
		{key: "4", label: "単語を削除", color: colorRed},
		{key: "5", label: "終了", color: colorMuted},
	}
	selected := 0

	for {
		updatedCards, err := loadCards()
		if err != nil {
			return err
		}
		cards = updatedCards

		choice := items[selected].key
		if isInteractiveTerminal() {
			render := func(current int) {
				clearScreen()
				printHero("Tangocho", fmt.Sprintf("%d cards saved locally", len(cards)))
				printMenu(cards, items, current)
			}
			render(selected)

			next, err := readMenuSelection(selected, len(items), render)
			if err != nil {
				return err
			}
			selected = next
			choice = items[selected].key
		} else {
			clearScreen()
			printHero("Tangocho", fmt.Sprintf("%d cards saved locally", len(cards)))
			printMenu(cards, items, selected)
			fmt.Print(styled(colorCyan, "Select") + " > ")
			choice, err = readLine(reader)
			if err != nil {
				return err
			}
		}

		switch choice {
		case "1":
			clearScreen()
			if err := addCardInteractive(cards); err != nil {
				return err
			}
		case "2":
			clearScreen()
			printCards(cards)
			waitForEnter(reader, "メニューに戻るには Enter")
		case "3":
			clearScreen()
			if err := runFlipCards(cards); err != nil {
				return err
			}
			waitForEnter(reader, "メニューに戻るには Enter")
		case "4":
			clearScreen()
			if err := deleteCardInteractive(cards); err != nil {
				return err
			}
		case "5":
			clearScreen()
			fmt.Println(panel("Goodbye", []string{
				"単語帳を終了します。",
			}, colorMuted))
			return nil
		default:
			fmt.Println(panel("Input Error", []string{
				"1-5 のいずれかを選んでください。",
			}, colorRed))
			waitForEnter(reader, "続けるには Enter")
		}
	}
}

func addCardInteractive(cards []Card) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(panel("Add Card", []string{
		"新しい英単語と意味を登録します。",
	}, colorGreen))

	fmt.Print(styled(colorCyan, "English") + " > ")
	english, err := readLine(reader)
	if err != nil {
		return err
	}
	if english == "" {
		return errors.New("英単語は空にできません")
	}

	fmt.Print(styled(colorCyan, "Japanese") + " > ")
	japanese, err := readLine(reader)
	if err != nil {
		return err
	}
	if japanese == "" {
		return errors.New("意味は空にできません")
	}

	card := Card{
		English:   english,
		Japanese:  japanese,
		CreatedAt: time.Now(),
	}

	cards = append(cards, card)
	if err := saveCards(cards); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(panel("Saved", []string{
		fmt.Sprintf("%s  ->  %s", card.English, card.Japanese),
	}, colorGreen))
	waitForEnter(reader, "続けるには Enter")
	return nil
}

func deleteCardInteractive(cards []Card) error {
	reader := bufio.NewReader(os.Stdin)

	if len(cards) == 0 {
		fmt.Println(panel("No Cards", []string{
			"削除できる単語はありません。",
		}, colorYellow))
		waitForEnter(reader, "メニューに戻るには Enter")
		return nil
	}

	printCards(cards)

	fmt.Print(styled(colorRed, "削除する番号 (空でキャンセル)") + " > ")
	input, err := readLine(reader)
	if err != nil {
		return err
	}
	if input == "" {
		return nil
	}

	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > len(cards) {
		fmt.Println(panel("Invalid", []string{
			fmt.Sprintf("1〜%d の番号を入力してください。", len(cards)),
		}, colorRed))
		waitForEnter(reader, "続けるには Enter")
		return nil
	}

	target := cards[n-1]
	fmt.Println()
	fmt.Println(panel("Confirm Delete", []string{
		fmt.Sprintf("削除: %s", styled(colorBold, target.English)),
		fmt.Sprintf("     %s", styled(colorMuted, target.Japanese)),
		"",
		styled(colorYellow, "本当に削除しますか？ [y/n]"),
	}, colorRed))
	fmt.Print(styled(colorRed, "Confirm") + " > ")
	confirm, err := readLine(reader)
	if err != nil {
		return err
	}

	if strings.ToLower(confirm) != "y" {
		fmt.Println(panel("Cancelled", []string{"削除をキャンセルしました。"}, colorMuted))
		waitForEnter(reader, "続けるには Enter")
		return nil
	}

	newCards := make([]Card, 0, len(cards)-1)
	newCards = append(newCards, cards[:n-1]...)
	newCards = append(newCards, cards[n:]...)
	if err := saveCards(newCards); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(panel("Deleted", []string{
		fmt.Sprintf("「%s」を削除しました。", target.English),
	}, colorRed))
	waitForEnter(reader, "メニューに戻るには Enter")
	return nil
}

func printCards(cards []Card) {
	if len(cards) == 0 {
		fmt.Println(panel("No Cards", []string{
			"単語はまだ登録されていません。",
		}, colorYellow))
		return
	}

	printHero("Cards", fmt.Sprintf("%d cards", len(cards)))
	for i, card := range cards {
		lines := []string{
			fmt.Sprintf("%s %s", badge(strconv.Itoa(i+1), colorCyan), styled(colorBold, card.English)),
			fmt.Sprintf("   %s", styled(colorMuted, card.Japanese)),
		}
		fmt.Println(panel("", lines, ""))
	}
}

// runFlipCards uses raw mode so every action requires just one keypress:
//   Space / Enter      — flip card
//   Y / Space / Enter / → — mark correct
//   N / ←             — mark incorrect
//   Q                  — quit session
func runFlipCards(cards []Card) error {
	if len(cards) == 0 {
		fmt.Println(panel("No Cards", []string{
			"単語はまだ登録されていません。先に追加してください。",
		}, colorYellow))
		return nil
	}

	shuffled := append([]Card(nil), cards...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	total := len(shuffled)
	correct := 0

	state, err := enableRawMode()
	if err != nil {
		return err
	}
	defer restoreTerminal(state)

outer:
	for i, card := range shuffled {
		// --- question phase ---
		clearScreen()
		printFlipCard(card.English, "", i+1, total, correct, false)

		for {
			key, err := readKey()
			if err != nil {
				break outer
			}
			switch key {
			case "enter", "space":
				goto showAnswer
			case "q":
				clearScreen()
				printSessionResult(correct, i)
				return nil
			}
		}

	showAnswer:
		// --- answer phase ---
		clearScreen()
		printFlipCard(card.English, card.Japanese, i+1, total, correct, true)

		for {
			key, err := readKey()
			if err != nil {
				break outer
			}
			switch key {
			case "y", "enter", "space", "right":
				correct++
				continue outer
			case "n", "left":
				continue outer
			case "q":
				clearScreen()
				printSessionResult(correct, i+1)
				return nil
			}
		}
	}

	clearScreen()
	printFinalScore(correct, total)
	return nil
}

// printFlipCard renders the flashcard UI.
// flipped=false shows only the English word; flipped=true reveals the Japanese answer.
func printFlipCard(english, japanese string, num, total, correct int, flipped bool) {
	cardInner := panelWidth - 2 // visible content width inside card borders (52)
	dashes := strings.Repeat("─", cardInner)

	bar := progressBar(num-1, total, 18)
	wrong := (num - 1) - correct
	subtitle := fmt.Sprintf("%s %d/%d   %s %d  %s %d",
		bar, num, total,
		styled(colorGreen, "✓"), correct,
		styled(colorRed, "✗"), wrong,
	)
	printHero("Flip Cards", subtitle)

	emptyRow := func(accent string) string {
		return "  " + styled(accent, "│") + strings.Repeat(" ", cardInner) + styled(accent, "│")
	}
	contentRow := func(accent, content string) string {
		return "  " + styled(accent, "│") + padRight(content, cardInner) + styled(accent, "│")
	}

	if !flipped {
		accent := colorCyan
		word := centerText(styled(colorBold+colorCyan, strings.ToUpper(english)), cardInner)

		fmt.Println("  " + styled(accent, "┌"+dashes+"┐"))
		fmt.Println(emptyRow(accent))
		fmt.Println(contentRow(accent, word))
		fmt.Println(emptyRow(accent))
		fmt.Println("  " + styled(accent, "└"+dashes+"┘"))
		fmt.Println()
		fmt.Printf("  %s フリップ    %s 終了\n",
			styled(colorMuted, "Space/Enter →"),
			styled(colorMuted, "Q →"),
		)
	} else {
		accent := colorGreen
		question := " " + styled(colorMuted, strings.ToUpper(english))
		answer := centerText(styled(colorBold+colorGreen, japanese), cardInner)

		fmt.Println("  " + styled(accent, "┌"+dashes+"┐"))
		fmt.Println(contentRow(accent, question))
		fmt.Println("  " + styled(accent, "├"+dashes+"┤"))
		fmt.Println(emptyRow(accent))
		fmt.Println(contentRow(accent, answer))
		fmt.Println(emptyRow(accent))
		fmt.Println("  " + styled(accent, "└"+dashes+"┘"))
		fmt.Println()
		fmt.Printf("  %s 知ってた    %s 知らなかった    %s 終了\n",
			styled(colorGreen, "Y/Space/→ →"),
			styled(colorRed, "N/← →"),
			styled(colorMuted, "Q →"),
		)
	}
}

func printSessionResult(correct, done int) {
	if done == 0 {
		fmt.Println(panel("Session Ended", []string{"学習を終了します。"}, colorMuted))
		return
	}
	pct := correct * 100 / done
	fmt.Println(panel("Session Ended", []string{
		fmt.Sprintf("正解: %s / %d  (%d%%)", styled(colorBold+colorGreen, strconv.Itoa(correct)), done, pct),
		"学習を終了します。",
	}, colorMuted))
}

func printFinalScore(correct, total int) {
	pct := 0
	if total > 0 {
		pct = correct * 100 / total
	}
	bar := progressBar(correct, total, panelWidth-22)
	grade := gradeLabel(pct)
	fmt.Println(panel("Result", []string{
		fmt.Sprintf("正解: %s / %d  (%d%%)", styled(colorBold+colorGreen, strconv.Itoa(correct)), total, pct),
		fmt.Sprintf("%s  %s", bar, grade),
		"",
		"すべてのカードを確認しました。",
	}, colorGreen))
}

func gradeLabel(pct int) string {
	switch {
	case pct >= 90:
		return styled(colorGreen+colorBold, "Excellent!")
	case pct >= 70:
		return styled(colorCyan+colorBold, "Good job!")
	case pct >= 50:
		return styled(colorYellow+colorBold, "Keep going!")
	default:
		return styled(colorRed+colorBold, "Keep studying!")
	}
}

func loadCards() ([]Card, error) {
	path := dataFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Card{}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cards []Card
	if len(data) == 0 {
		return []Card{}, nil
	}
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return cards, nil
}

func saveCards(cards []Card) error {
	data, err := json.MarshalIndent(cards, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode cards: %w", err)
	}
	path := dataFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cards: %w", err)
	}
	return nil
}

func dataFilePath() string {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		return dataFileName
	}

	return filepath.Join(dataDir, "vocabulary", dataFileName)
}

func readLine(reader *bufio.Reader) (string, error) {
	text, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func printHelp() {
	fmt.Println("使い方:")
	fmt.Println("  vocabulary          対話メニューを開く")
	fmt.Println("  vocabulary add      単語を追加")
	fmt.Println("  vocabulary list     単語一覧を見る")
	fmt.Println("  vocabulary flip     フリップカードで学習")
	fmt.Printf("  データ保存先: %s\n", dataFilePath())
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, styled(colorRed, "Error:"), err)
	fmt.Println()
	printHelp()
	os.Exit(1)
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// printHero renders the top banner. subtitle may contain ANSI codes.
func printHero(title, subtitle string) {
	fmt.Println(styled(colorCyan+colorBold, "╭"+strings.Repeat("─", panelWidth)+"╮"))
	fmt.Println(styled(colorCyan+colorBold, "│") + padRight(" "+styled(colorBold, title), panelWidth) + styled(colorCyan+colorBold, "│"))
	fmt.Println(styled(colorMuted, "│") + padRight(" "+subtitle, panelWidth) + styled(colorMuted, "│"))
	fmt.Println(styled(colorCyan+colorBold, "╰"+strings.Repeat("─", panelWidth)+"╯"))
	fmt.Println()
}

func printMenu(cards []Card, items []menuItem, selected int) {
	lines := []string{}
	for i, item := range items {
		cursor := " "
		label := fmt.Sprintf("%s %s", badge(item.key, item.color), item.label)
		if i == selected {
			cursor = "›"
			label = styled(item.color+colorBold, label+"  ")
		}
		lines = append(lines, fmt.Sprintf("%s %s", styled(item.color, cursor), label))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s ↑ ↓ で移動 / Enter で決定", styled(colorMuted, "Hint")))
	lines = append(lines, fmt.Sprintf("%s 保存先: %s", styled(colorMuted, "Data"), dataFilePath()))
	if len(cards) == 0 {
		lines = append(lines, styled(colorYellow, "まだ単語はありません。まずは 1 で追加してください。"))
	}
	fmt.Println(panel("Menu", lines, ""))
}

func waitForEnter(reader *bufio.Reader, label string) {
	fmt.Println()
	fmt.Print(styled(colorMuted, label) + " > ")
	_, _ = readLine(reader)
}

func styled(color, text string) string {
	return color + text + colorReset
}

func badge(text, color string) string {
	return styled(color+colorBold, "["+text+"]")
}

func panel(title string, lines []string, accent string) string {
	width := panelWidth
	borderColor := accent
	if borderColor == "" {
		borderColor = colorMuted
	}

	var b strings.Builder
	b.WriteString(styled(borderColor, "┌"+strings.Repeat("─", width)+"┐"))
	b.WriteString("\n")
	if title != "" {
		header := fmt.Sprintf("  %s", title)
		b.WriteString(styled(borderColor, "│"))
		b.WriteString(padRight(styled(colorBold, header), width))
		b.WriteString(styled(borderColor, "│"))
		b.WriteString("\n")
		b.WriteString(styled(borderColor, "├"+strings.Repeat("─", width)+"┤"))
		b.WriteString("\n")
	}
	for _, line := range lines {
		b.WriteString(styled(borderColor, "│"))
		b.WriteString(padRight(" "+line, width))
		b.WriteString(styled(borderColor, "│"))
		b.WriteString("\n")
	}
	b.WriteString(styled(borderColor, "└"+strings.Repeat("─", width)+"┘"))
	return b.String()
}

func padRight(text string, width int) string {
	visible := visibleLen(text)
	if visible >= width {
		return trimDisplayWidth(text, width)
	}
	return text + strings.Repeat(" ", width-visible)
}

func visibleLen(text string) int {
	length := 0
	inEscape := false
	for _, r := range text {
		switch {
		case r == '\033':
			inEscape = true
		case inEscape && r == 'm':
			inEscape = false
		case !inEscape:
			length += runeCellWidth(r)
		}
	}
	return length
}

func progressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}
	filled := current * width / total
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return styled(colorGreen, strings.Repeat("█", filled)) + styled(colorMuted, strings.Repeat("░", width-filled))
}

func centerText(text string, width int) string {
	if visibleLen(text) >= width {
		return trimDisplayWidth(text, width)
	}
	left := (width - visibleLen(text)) / 2
	return strings.Repeat(" ", left) + text
}

func trimDisplayWidth(text string, width int) string {
	if width <= 0 {
		return ""
	}

	var b strings.Builder
	visible := 0
	inEscape := false
	hasEscape := false

	for _, r := range text {
		switch {
		case r == '\033':
			inEscape = true
			hasEscape = true
			b.WriteRune(r)
		case inEscape:
			b.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
		default:
			w := runeCellWidth(r)
			if visible+w > width {
				if hasEscape && !strings.HasSuffix(b.String(), colorReset) {
					b.WriteString(colorReset)
				}
				return b.String()
			}
			b.WriteRune(r)
			visible += w
		}
	}

	if hasEscape && !strings.HasSuffix(b.String(), colorReset) {
		b.WriteString(colorReset)
	}
	return b.String()
}

func runeCellWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r < 32 || (r >= 0x7f && r < 0xa0):
		return 0
	case unicode.Is(unicode.Mn, r):
		return 0
	case isWideRune(r):
		return 2
	default:
		return 1
	}
}

func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115f:
		return true
	case r >= 0x2329 && r <= 0x232a:
		return true
	case r >= 0x2e80 && r <= 0xa4cf:
		return true
	case r >= 0xac00 && r <= 0xd7a3:
		return true
	case r >= 0xf900 && r <= 0xfaff:
		return true
	case r >= 0xfe10 && r <= 0xfe19:
		return true
	case r >= 0xfe30 && r <= 0xfe6f:
		return true
	case r >= 0xff01 && r <= 0xff60:
		return true
	case r >= 0xffe0 && r <= 0xffe6:
		return true
	case r >= 0x1f300 && r <= 0x1faff:
		return true
	default:
		return false
	}
}

func isInteractiveTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func readMenuSelection(current, total int, render func(int)) (int, error) {
	state, err := enableRawMode()
	if err != nil {
		return current, err
	}
	defer func() {
		_ = restoreTerminal(state)
		fmt.Println()
	}()

	selected := current
	for {
		key, err := readKey()
		if err != nil {
			return current, err
		}

		switch key {
		case "up":
			selected--
			if selected < 0 {
				selected = total - 1
			}
			render(selected)
		case "down":
			selected++
			if selected >= total {
				selected = 0
			}
			render(selected)
		case "enter":
			return selected, nil
		case "1", "2", "3", "4", "5":
			index := int(key[0] - '1')
			if index >= 0 && index < total {
				render(index)
				return index, nil
			}
		case "q":
			render(total - 1)
			return total - 1, nil
		}
	}
}

func enableRawMode() (string, error) {
	stateCmd := exec.Command("stty", "-g")
	stateCmd.Stdin = os.Stdin
	stateBytes, err := stateCmd.Output()
	if err != nil {
		tty, ttyErr := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if ttyErr != nil {
			return "", fmt.Errorf("failed to read terminal state: %w", err)
		}
		defer tty.Close()

		stateCmd = exec.Command("stty", "-g")
		stateCmd.Stdin = tty
		stateBytes, err = stateCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to read terminal state: %w", err)
		}
	}
	state := strings.TrimSpace(string(stateBytes))

	cmd := exec.Command("stty", "-icanon", "-echo", "min", "1", "time", "0")
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		tty, ttyErr := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if ttyErr != nil {
			return "", fmt.Errorf("failed to enable raw mode: %w", err)
		}
		defer tty.Close()

		cmd = exec.Command("stty", "-icanon", "-echo", "min", "1", "time", "0")
		cmd.Stdin = tty
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to enable raw mode: %w", err)
		}
	}
	return state, nil
}

func restoreTerminal(state string) error {
	cmd := exec.Command("stty", state)
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err == nil {
		return nil
	}

	tty, ttyErr := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if ttyErr != nil {
		return ttyErr
	}
	defer tty.Close()

	cmd = exec.Command("stty", state)
	cmd.Stdin = tty
	return cmd.Run()
}

func readKey() (string, error) {
	buf := make([]byte, 3)

	n, err := os.Stdin.Read(buf[:1])
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", io.EOF
	}

	switch buf[0] {
	case '\r', '\n':
		return "enter", nil
	case ' ':
		return "space", nil
	case 'q', 'Q':
		return "q", nil
	case 'y', 'Y':
		return "y", nil
	case 'n', 'N':
		return "n", nil
	case '1', '2', '3', '4', '5':
		return string(buf[0]), nil
	case 27:
		if _, err := os.Stdin.Read(buf[1:2]); err != nil {
			return "", err
		}
		if _, err := os.Stdin.Read(buf[2:3]); err != nil {
			return "", err
		}
		if buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return "up", nil
			case 'B':
				return "down", nil
			case 'C':
				return "right", nil
			case 'D':
				return "left", nil
			}
		}
	}

	return "", nil
}
