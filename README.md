# dokuha(読破)

dokuha(読破) is a TUI application to inspect Japanese text and export unknown vocabularies for language learner.

It is powered by [kagome](https://github.com/ikawaha/kagome), a self-contained japanese morphological analyzer written in pure Go, and uses [termui](https://github.com/gizak/termui) to implement user interface.

<p align="center">
    <img width="2492" alt="dokuha 読破 preview" src="https://user-images.githubusercontent.com/1995921/62272704-a3dfbb00-b476-11e9-9df1-93d1e014fb6a.gif">
</p>


## Install


```shell
go get github.com/masakichi/dokuha
```

## Configuration

TBD


## Keymap

```
- q or <C-c>: quit
- 1, 2, 3, 4, 5: switch to different widget
- k or <Up>: up
- j or <Down>: down
- <C-u>: half page up
- <C-d>: half page down
- <C-f>: page down
- <C-b>: page up
- T: toggle display all words or unknown words only
- gg: jump to top
- G: jump to bottom
- <Space>: select or deselect a word

```

## Usage

TBD
