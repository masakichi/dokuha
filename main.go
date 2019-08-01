package main

import (
	"crypto/sha256"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	"github.com/ikawaha/kagome/tokenizer"
	"github.com/masakichi/dokuha/utils"
	_ "github.com/mattn/go-sqlite3"
)

const (
	appName = "dokuha"
)

var (
	kagome tokenizer.Tokenizer

	grid *ui.Grid

	miniTokenList  []miniToken
	nounTokenList  []*miniToken
	verbTokenList  []*miniToken
	adjTokenList   []*miniToken
	otherTokenList []*miniToken

	nounWidget         *widgets.List
	verbWidget         *widgets.List
	adjWidget          *widgets.List
	otherWidget        *widgets.List
	selectedWordWidget *widgets.List
	allListWidgets     []*widgets.List
	focusedWidget      *widgets.List

	knownWordList     = []string{}
	ankiWordList      = []string{}
	displayKnownWords = true
)

type miniToken struct {
	Lemma string
	Pos   string
	Yomi  string
	Count int
	Goshu string
}

func (t *miniToken) SimpleLemma() string {
	return strings.Split(t.Lemma, "-")[0]
}

func tokenFilter(tokens []miniToken, f func(token miniToken) bool) []*miniToken {
	results := make([]*miniToken, 0)
	for _, v := range tokens {
		if f(v) {
			func(v miniToken) {
				results = append(results, &v)
			}(v)
		}
	}
	return results
}

func analyzeText(s string) {
	lemmaMap := map[string]*miniToken{}
	tokens := kagome.Analyze(s, tokenizer.Normal)
	for _, token := range tokens {
		if token.Class == tokenizer.DUMMY || token.Class == tokenizer.UNKNOWN || token.Pos() == "空白" {
			continue
		}
		if strings.Contains(token.Pos(), "記号") {
			continue
		}
		features := token.Features()
		if len(features) < 7 {
			fmt.Println(token)
			continue
		}
		// features 内容はこっちに参照 https://hayashibe.jp/tr/mecab/dictionary/unidic/field
		lemma := features[7]
		lemmaRunes := []rune(lemma)
		// filter single hiragana or katakana
		if len(lemmaRunes) == 1 && (unicode.In(lemmaRunes[0], unicode.Hiragana) || unicode.In(lemmaRunes[0], unicode.Katakana)) {
			continue
		}
		if _, ok := lemmaMap[lemma]; ok {
			lemmaMap[lemma].Count++
		} else {
			lemmaMap[lemma] = &miniToken{lemma, token.Pos(), features[6], 1, features[12]}
		}
	}
	for _, v := range lemmaMap {
		miniTokenList = append(miniTokenList, *v)
	}
	sort.Slice(miniTokenList, func(i, j int) bool { return miniTokenList[i].Count > miniTokenList[j].Count })
}

func initKagome() {
	kagome = tokenizer.NewWithDic(tokenizer.SysDicUni())
}

func setupWidgets() {
	nounTokenList = tokenFilter(miniTokenList, func(t miniToken) bool { return strings.Contains(t.Pos, "名詞") })
	verbTokenList = tokenFilter(miniTokenList, func(t miniToken) bool { return t.Pos == "動詞" })
	adjTokenList = tokenFilter(miniTokenList, func(t miniToken) bool { return t.Pos == "形容詞" || t.Pos == "形状詞" })
	otherTokenList = tokenFilter(miniTokenList, func(t miniToken) bool {
		return t.Pos != "形容詞" && t.Pos != "形状詞" && t.Pos != "動詞" && !strings.Contains(t.Pos, "名詞")
	})

	nounWidget = newTokenWidget("名詞")
	verbWidget = newTokenWidget("動詞")
	adjWidget = newTokenWidget("形容形状詞")
	otherWidget = newTokenWidget("その他")
	setupWidgetRows()

	selectedWordWidget = widgets.NewList()
	selectedWordWidget.Title = "知ってる単語"
	selectedWordWidget.Rows = []string{}
	selectedWordWidget.WrapText = true

	allListWidgets = []*widgets.List{nounWidget, verbWidget, adjWidget, otherWidget, selectedWordWidget}

	// focus nounWidget by default
	focusWidget("1")
}

func setupWidgetRows() {
	applyRowsToWidget(nounWidget, nounTokenList, false)
	applyRowsToWidget(verbWidget, verbTokenList, false)
	applyRowsToWidget(adjWidget, adjTokenList, false)
	applyRowsToWidget(otherWidget, otherTokenList, true)
}

func setupKnownWordList() {
	bytes, err := ioutil.ReadFile("known_words.txt")
	if err == nil {
		knownWordList = strings.Split(string(bytes), "\n")
	}
}

func setupGrid() {
	grid = ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(
			1,
			ui.NewCol(1.0/3,
				ui.NewRow(1.0/2, nounWidget),
				ui.NewRow(1.0/2, verbWidget),
			),
			ui.NewCol(1.0/3,
				ui.NewRow(1.0/2, adjWidget),
				ui.NewRow(1.0/2, otherWidget),
			),
			ui.NewCol(1.0/3,
				ui.NewRow(1, selectedWordWidget),
			),
		),
	)
}

func newTokenWidget(title string) *widgets.List {
	l := widgets.NewList()
	l.Title = title
	l.WrapText = true
	return l
}

func applyRowsToWidget(w *widgets.List, tokens []*miniToken, verbose bool) {
	w.Rows = func(tl []*miniToken) []string {
		rv := make([]string, 0)
		for _, t := range tl {
			if !displayKnownWords {
				// do not append t to rv if t.Lemma in knownWordList
				if func(t *miniToken) bool {
					for _, v := range append(knownWordList, ankiWordList...) {
						if v == t.SimpleLemma() {
							return true
						}
					}
					return false
				}(t) {
					continue
				}
			}
			if verbose {
				rv = append(rv, fmt.Sprintf("%s (%s・%s・%d)", t.SimpleLemma(), t.Pos, t.Goshu, t.Count))
			} else {
				rv = append(rv, fmt.Sprintf("%s (%s・%d)", t.SimpleLemma(), t.Goshu, t.Count))
			}
		}
		return rv
	}(tokens)
}

func focusWidget(opt string) {
	idx, err := strconv.Atoi(opt)
	if err != nil || idx-1 > len(allListWidgets) {
		return
	}
	for _, w := range allListWidgets {
		w.TextStyle = ui.NewStyle(ui.ColorWhite)
		w.BorderStyle = ui.NewStyle(ui.ColorWhite)
	}
	focusedWidget = allListWidgets[idx-1]
	focusedWidget.BorderStyle = ui.NewStyle(ui.ColorCyan)
	focusedWidget.TextStyle = ui.NewStyle(ui.ColorGreen)
}

func toggleWord(_w string) {
	w := strings.Split(_w, " ")[0]
	rowsPtr := &(selectedWordWidget.Rows)
	for idx, _v := range *rowsPtr {
		v := strings.Split(_v, " ")[0]
		if v == w {
			*rowsPtr = append((*rowsPtr)[:idx], (*rowsPtr)[idx+1:]...)
			return
		}
	}
	*rowsPtr = append(*rowsPtr, w)
}

func saveSelectedWords() {
	if len(selectedWordWidget.Rows) == 0 {
		return
	}
	knownWordMap := make(map[string]struct{})
	for _, w := range knownWordList {
		knownWordMap[w] = struct{}{}
	}
	needAppendWordList := make([]string, 0)
	for _, knownWord := range selectedWordWidget.Rows {
		if _, ok := knownWordMap[knownWord]; !ok {
			needAppendWordList = append(needAppendWordList, knownWord)
		}
	}
	f, _ := os.OpenFile("known_words.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString(strings.Join(needAppendWordList, "\n"))
	f.WriteString("\n")
}

func saveUnknownWords() {
	knownWordMap := make(map[string]struct{})
	for _, wl := range [][]string{knownWordList, ankiWordList, selectedWordWidget.Rows} {
		for _, w := range wl {
			knownWordMap[w] = struct{}{}
		}
	}
	unknownWords := make([]string, 0)
	for _, t := range miniTokenList {
		if _, ok := knownWordMap[t.SimpleLemma()]; !ok {
			unknownWords = append(unknownWords, t.SimpleLemma())
		}
	}
	ioutil.WriteFile("unknown_words.txt", []byte(strings.Join(unknownWords, "\n")+"\n"), 0644)
}

func hackScrollCrash(scrollFunc func()) {
	if focusedWidget.SelectedRow > len(focusedWidget.Rows)-1 {
		return
	}
	scrollFunc()
}

func eventLoop() {
	previousKey := ""
	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "1", "2", "3", "4", "5":
			focusWidget(e.ID)
		case "q", "<C-c>":
			saveSelectedWords()
			saveUnknownWords()
			return
		case "j", "<Down>":
			hackScrollCrash(focusedWidget.ScrollDown)
		case "k", "<Up>":
			focusedWidget.ScrollUp()
		case "<C-d>":
			focusedWidget.ScrollHalfPageDown()
		case "<C-u>":
			focusedWidget.ScrollHalfPageUp()
		case "<C-f>":
			focusedWidget.ScrollPageDown()
		case "<C-b>":
			focusedWidget.ScrollPageUp()
		case "T":
			displayKnownWords = !displayKnownWords
			setupWidgetRows()
		case "g":
			if previousKey == "g" {
				focusedWidget.ScrollTop()
			}
		case "G":
			focusedWidget.ScrollBottom()
		case "<Space>":
			if focusedWidget.SelectedRow <= len(focusedWidget.Rows)-1 {
				w := focusedWidget.Rows[focusedWidget.SelectedRow]
				toggleWord(w)
			}
		case "<Resize>":
			payload := e.Payload.(ui.Resize)
			termWidth, termHeight := payload.Width, payload.Height
			grid.SetRect(0, 0, termWidth, termHeight)
		}

		if previousKey == "g" {
			previousKey = ""
		} else {
			previousKey = e.ID
		}

		ui.Render(grid)
	}
}

func syncAnkiWords() {
	//TODO(Gimo): make it configurable
	dbPath := filepath.Join("/home/yuanji/.local/share/Anki2/Yuanji/", "collection.anki2")
	utils.AnkiDB, _ = sql.Open("sqlite3", dbPath)
	deckID := utils.GetAnkiDeckID("日本語")
	ankiWordList = utils.GetWordsByAnkiDeckID(deckID)
}

func main() {

	if len(os.Args) < 2 {
		log.Fatal("please give me a txt file")
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s: A TUI Japanese text inspection tool\n", appName)
		fmt.Fprintf(os.Stderr, "Usage: %s %s\n\n", os.Args[0], "filename.txt")
		flag.PrintDefaults()
	}

	flag.Parse()
	bytes, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	cacheName := fmt.Sprintf("%x", sha256.Sum256(bytes))
	utils.CacheDir = utils.GetCacheDir(appName)
	utils.EnsureDir(utils.CacheDir)

	err = utils.LoadCache(cacheName, &miniTokenList)
	if err != nil {
		initKagome()
		analyzeText(string(bytes))
		utils.SetCache(cacheName, miniTokenList)
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	setupKnownWordList()
	syncAnkiWords()
	setupWidgets()
	setupGrid()
	ui.Render(grid)

	eventLoop()

}
