package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	"github.com/ikawaha/kagome/tokenizer"
)

var (
	kagome tokenizer.Tokenizer

	grid *ui.Grid

	miniTokenList  []*miniToken
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
)

type miniToken struct {
	Lemma string
	Pos   string
	Count int
	Goshu string
}

func tokenFilter(tokens []*miniToken, f func(token *miniToken) bool) []*miniToken {
	results := make([]*miniToken, 0)
	for _, v := range tokens {
		if f(v) {
			results = append(results, v)
		}
	}
	return results
}

func analyzeText(s string) {
	lemmaMap := map[string]*miniToken{}
	tokens := kagome.Analyze(s, tokenizer.Normal)
	for _, token := range tokens {
		if token.Class == tokenizer.DUMMY || token.Class == tokenizer.UNKNOWN {
			// BOS: Begin Of Sentence, EOS: End Of Sentence.
			//fmt.Printf("%s\n", token.Surface)
			continue
		}
		features := token.Features()
		if len(features) < 7 {
			fmt.Println(token)
			continue
		}
		lemma := features[7]
		if _, ok := lemmaMap[lemma]; ok {
			lemmaMap[lemma].Count++
		} else {
			lemmaMap[lemma] = &miniToken{lemma, token.Pos(), 1, features[12]}
		}
	}
	for _, v := range lemmaMap {
		miniTokenList = append(miniTokenList, v)
	}
	sort.Slice(miniTokenList, func(i, j int) bool { return miniTokenList[i].Count > miniTokenList[j].Count })
}

func initKagome() {
	kagome = tokenizer.NewWithDic(tokenizer.SysDicUni())
}

func setupWidgets() {
	nounTokenList = tokenFilter(miniTokenList, func(t *miniToken) bool { return strings.Contains(t.Pos, "名詞") })
	verbTokenList = tokenFilter(miniTokenList, func(t *miniToken) bool { return t.Pos == "動詞" })
	adjTokenList = tokenFilter(miniTokenList, func(t *miniToken) bool { return t.Pos == "形容詞" || t.Pos == "形状詞" })
	otherTokenList = tokenFilter(miniTokenList, func(t *miniToken) bool {
		return t.Pos != "形容詞" && t.Pos != "形状詞" && t.Pos != "動詞" && !strings.Contains(t.Pos, "記号") && !strings.Contains(t.Pos, "名詞")
	})

	nounWidget = newTokenWidget("名詞", nounTokenList, false)
	verbWidget = newTokenWidget("動詞", verbTokenList, false)
	adjWidget = newTokenWidget("形容形状詞", adjTokenList, false)
	otherWidget = newTokenWidget("その他", otherTokenList, true)

	selectedWordWidget = widgets.NewList()
	selectedWordWidget.Title = "単語帳"
	selectedWordWidget.Rows = []string{}
	selectedWordWidget.WrapText = true

	allListWidgets = []*widgets.List{nounWidget, verbWidget, adjWidget, otherWidget, selectedWordWidget}

	// focus nounWidget by default
	focusWidget("1")
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

func newTokenWidget(title string, tokens []*miniToken, verbose bool) *widgets.List {
	l := widgets.NewList()
	l.Title = title
	l.Rows = func(tl []*miniToken) []string {
		rv := make([]string, 0)
		for _, t := range tl {
			if verbose {
				rv = append(rv, fmt.Sprintf("%s (%s・%s・%d)", t.Lemma, t.Pos, t.Goshu, t.Count))
			} else {
				rv = append(rv, fmt.Sprintf("%s (%s・%d)", t.Lemma, t.Goshu, t.Count))
			}
		}
		return rv
	}(tokens)
	l.WrapText = true
	return l
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

func toggleWord(w string) {
	rowsPtr := &(selectedWordWidget.Rows)
	for idx, v := range *rowsPtr {
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
	bytes := []byte(strings.Join(selectedWordWidget.Rows, "\n"))
	ioutil.WriteFile("words.txt", bytes, 0644)
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
			return
		case "j", "<Down>":
			focusedWidget.ScrollDown()
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

func main() {

	fileName := flag.String("f", "", "file name")
	flag.Parse()
	file, err := os.Open(*fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)

	initKagome()
	analyzeText(string(bytes))

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	setupWidgets()
	setupGrid()
	ui.Render(grid)

	eventLoop()

}
