package main

import (
  "testing"
  "fmt"
)

func surroundPara(str string) string {
  return "\n<p>" + str + "</p>\n"
}

func surroundArticle(str string) string {
  return "<article>\n" + str + "\n</article>"
}

func surroundArticlePara(str string) string {
  return surroundArticle(surroundPara(str))
}

func TestHeadingsCorrect(t* testing.T) {
  fmt.Println("TEST:: Running TestHeadingsCorrect")
  // h1
  parsed_h1 := ProcessMD("# h1\n")
  if parsed_h1 != surroundArticle("\n<h1>h1</h1>\n") {
    t.Fatalf("ERROR:: Invalid h1 header after parsing\n%s\n", parsed_h1)
  }
  // h6
  parsed_h6 := ProcessMD("###### h6\n")
  if parsed_h6 != surroundArticle("\n<h6>h6</h6>\n") {
    t.Fatalf("ERROR:: Invalid h6 header after parsing\n%s\n", parsed_h6)
  }
  // multiple # chars
  multi_h := ProcessMD("## h2 with # # and ###\n")
  if multi_h != surroundArticle("\n<h2>h2 with # # and ###</h2>\n") {
    t.Fatalf("ERROR:: Invalid header with multiple # after parsing\n%s\n", multi_h)
  }
}

func _TestHeadingsIncorrect(t* testing.T) {
  fmt.Println("TEST:: Running TestHeadingsIncorrect")
  // no space after header tag
  parsed_h1 := ProcessMD("#h1\n")
  if parsed_h1 != surroundArticle("\n<p>#h1</p>\n") {
    t.Fatalf("ERROR:: Unexpected handling of invalid h1\n%s\n", parsed_h1)
  }
  parsed_h1i := ProcessMD("#h1 actually *italics*\n")
  if parsed_h1i != surroundArticle("\n<p>#h1 actually <i>italics</i></p>\n") {
    t.Fatalf("ERROR:: Unexpected handling of invalid h1\n%s\n", parsed_h1i)
  }
}

func TestParagraph(t* testing.T) {
  fmt.Println("TEST:: Running TestParagraph")
  para := ProcessMD("test para")
  valid_str := surroundArticle("\n<p>test para</p>\n")
  if para != valid_str {
    t.Fatalf("ERROR:: Invalid parsing of paragraph\n%s\n", para)
  }
  para = ProcessMD("test para1\n\ntest para2")
  valid_str_body := `
<p>test para1
</p>

<p>test para2</p>
`
  if para != surroundArticle(valid_str_body) {
    t.Fatalf("ERROR:: Invalid parsing of multiple paragraphs\n%s\n", para)
  }
}

func _TestStylingsV1(t* testing.T) {
  fmt.Println("TEST:: Running TestStylingsV1")
  italic := ProcessMD("*italic text*")
  valid_str := surroundArticlePara("<i>italic text</i>")
  if italic != valid_str {
    t.Fatalf("ERROR:: Invalid parsing of italic text\n%s\n", italic)
  }
  bold := ProcessMD("**bold text**")
  valid_str = surroundArticlePara("<b>bold text</b>")
  if bold != valid_str {
    t.Fatalf("ERROR:: Invalid parsing of bold text\n%s\n", bold)
  }
  italicBold := ProcessMD("***italic bold text***")
  valid_str = surroundArticlePara("<i><b>italic bold text</b></i>")
  if italicBold != valid_str {
    t.Fatalf("ERROR:: Invalid parsing of italic bold text\n%s\n", italicBold)
  }
}

func _TestStylingsIncorrect(t* testing.T) {
  fmt.Println("TEST:: Running TestStylingsIncorrect")
  ivItalic := ProcessMD("* italic*")
  if ivItalic != surroundArticlePara("* italic*") {
    t.Fatalf("ERROR:: Invalid handling of invalid italic\n%s\n", ivItalic)
  }
  ivItalicNl := ProcessMD("*italic\n*")
  if ivItalicNl != surroundArticlePara("<i>italic\n</i>") {
    t.Fatalf("ERROR:: Invalid handling of invalid italic with newline\n%s\n", ivItalicNl)
  }
  ivBold := ProcessMD("**bold*")
  if ivBold != surroundArticlePara("**bold*") {
    t.Fatalf("ERROR:: Invalid handling of invalid bold\n%s\n", ivBold)
  }
  ivItalicBold := ProcessMD("***Italic Bold *")
  if ivItalicBold != surroundArticlePara("***Italic Bold *") {
    t.Fatalf("ERROR:: Invalid handling of invalid italic bold\n%s\n", ivItalicBold)
  }
}

func _TestLineBreak(t* testing.T) {
  fmt.Println("TEST:: Running TestLineBreak")
  lbSimple := ProcessMD("sample test  ")
  if lbSimple != surroundArticlePara("sample test <br />") {
    t.Fatalf("ERROR:: Invalid handling of simple line break\n%s\n", lbSimple)
  }

  lbItalic := ProcessMD("*sample test  *")
  if lbItalic != surroundArticlePara("<i>sample test <br /></i>") {
    t.Fatalf("ERROR:: Invalid handling of line break in Italic\n%s\n", lbItalic)
  }
}

