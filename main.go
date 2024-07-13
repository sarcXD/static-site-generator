package main

/*
@glossory:
- prefix write: this is a write we need to do before writing the currently checked element.
This is used primarily for paragraphs closing, as writing a new element, like a header,
usually means that we have to close a paragraph if one was being writting

- postfix write: this is a write we need to do after writing the currently buffered element.
A good example of this is a pagebreak, which can cause some buffers to be invalidated, like
the italicBold buffer. In that case, we have to first write the entire active buffer(s) and only
then do we write the pagebreak, else the pagebreak will be written before the active buffers,
which will be incorrect.

@important:
- add error validation, so whenever I am doing something undefined, I raise an error informing in detail,
what it is I am doing and why that is undefined behavior. I hate that markdown can be written on a whim
and you don't even know what you did incorrectly

@todo:
---
- md conversion
  * text formatting to work with newline and linebreak
	* link
	* ul
	* ol
- table? (probably a custom table)
- custom header
- custom components
@improvements:
- treat parsing error as values and do not exit the programme upon getting an error
  - continue parsing the string, checking for errors and notifying where there is an error
  - warn about errors but silently treat failing markdown as simple text
@inprogress:
@done:
- headings
- paragraphs
- linebreak
- text formatting
-- italic
-- bold
-- italicBold
*/

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
)

type pathState struct {
	// input items
	src_path  string
	src_files []string
	src_dirs  []string

	// output items
	dst_path string
}

var paraMap []string = []string{"<p>", "</p>"}
var italicBoldMap [][]string = [][]string{{"<i>", "</i>"}, {"<b>", "</b>"}, {"<i><b>", "</b></i>"}}

const (
	TokenNone = iota + 0
	TokenHeading
	TokenFormat
	TokenSpace
	TokenNewline
)

const (
	ParseSuccess = iota + 0
	ParseError
)

func Tokenize(ch rune) int {
	operation := TokenNone
	switch ch {
	case '#':
		operation = TokenHeading
	case '*':
		operation = TokenFormat
	case ' ':
		operation = TokenSpace
	case '\n':
		operation = TokenNewline
	default:
		operation = TokenNone
	}

	return operation
}

var hMap []string = []string{"h1", "h2", "h3", "h4", "h5", "h6"}

const (
	HmdNone = iota + 0
	HmdToken
	HmdText
	HmdDone
	HmdError
)

type ParsedToken struct {
	str           string
	pos           int
	row           int
	col           int
	statusCode    int
	statusMessage string
}

type paraState struct {
	begin    bool
	active   bool
	surround bool
	end      bool
}

type ParserState struct {
	inpStr      string
	outStr      string
	currPos     int
	writeBuffer string
	isSpace     bool
	isNewLine   bool
	para        paraState
}

type MdParser interface {
	printParseError(info ParsedToken) ParserState
	writeToOutputStr() ParserState
}

func ParseHeading(str string, pos int) (res ParsedToken) {
	res.str = ""
	res.pos = pos
	res.row = 1
	res.col = 0

	hInd := 0
	hStatus := HmdNone
	rawBuffer := ""
	parsedBuffer := ""
	i := pos
	for i = pos; i < len(str); i++ {
		res.col++
		ch := rune(str[i])
		switch ch {
		case '#':
			if hStatus < HmdText {
				// we are going through the list of # to see what kind of heading
				// this will be
				hInd++
				hStatus = HmdToken
				if hInd > len(hMap) {
					// we see more than 6 `#` characters. Those are invalid
					res.str = rawBuffer
					res.statusCode = ParseError
					res.statusMessage = "headings can only have at max 6 `#` characters to declare them."
					res.pos = i
					hStatus = HmdError
				}
			} else {
				// we are currently writing heading text and see another # character
				// that will just be writting inside the heading as is
				parsedBuffer += "#"
			}
			rawBuffer += "#"
		case ' ':
			if hStatus == HmdToken {
				// we were going through the list of headings and found a ` `
				// this means that text writing should begin now
				hStatus = HmdText
				parsedBuffer += "\n<" + hMap[hInd-1] + ">"
			} else {
				// in normal cases we will jsut copy the space
				parsedBuffer += " "
			}
			rawBuffer += " "
		case '\n':
			// a newline marks the end of a header
			// we will complete parsing and return
			rawBuffer += "\n"
			parsedBuffer += "</" + hMap[hInd-1] + ">\n"

			res.str = parsedBuffer
			res.statusCode = ParseSuccess
			res.pos = i
			hStatus = HmdDone
			res.row++
			res.col = 0
		default:
			// handle string
			if hStatus == HmdToken {
				// if we were going throuhg heading `#` characters and found a normal text character
				// that means that the heading is invalid
				res.str = rawBuffer
				res.statusCode = ParseError
				res.statusMessage = "An unsupported character was found directly after #. This is not valid"
				hStatus = HmdError
				// we want this to be re-evaluated after exiting since this will be treated as an independant character -
				// and in the event that is some other markdown character that needs evaluation, this ensures that we
				// do not skip it
				res.pos = i - 1
				break
			} else {
				// If the state is of writing, we will copy whatever character was found
				parsedBuffer += string(ch)
			}
			rawBuffer += string(ch)
		}
		if hStatus >= HmdDone {
			break
		}
	}
	if hStatus < HmdDone {
		res.str = rawBuffer
		res.statusCode = ParseError
		res.statusMessage = "Header was not terminated, as such it is invalid"
		res.pos = i
	}
	return res
}

func ClampFloor(val int, floor int) int {
	if val < floor {
		return floor
	}
	return val
}

func ClampCeil(val int, ceil int) int {
	if val > ceil {
		return ceil
	}
	return val
}

func (state ParserState) printParseError(info ParsedToken) {
	// print a detailed error message and exit the program
	fmt.Printf("ERROR:: %s.\nValue: ...%s > %s... \nLocation => line: %d, col: %d\n",
		info.statusMessage, 
    state.inpStr[ClampFloor(state.currPos-15, 0):info.pos], 
    state.inpStr[info.pos:ClampCeil(info.pos+15, len(state.inpStr)-1)],
		info.row, info.col)
}

func (state *ParserState) writeToOutputStr() {
	if state.para.end {
		if state.para.active {
			state.outStr += "</p>\n"
			state.para.active = false
			state.para.end = false
		}
	}
	if state.para.begin {
		state.outStr += "\n<p>"
		state.para.active = true
		state.para.begin = false
	}
	state.outStr += state.writeBuffer
	if state.para.surround {
		state.outStr += "</p>\n"
		state.para.surround = false
		state.para.active = false
		state.para.begin = false
		state.para.end = false
	}
}

func ProcessMD(str string) string {
	var state ParserState
	state.inpStr = str
	state.outStr = "<article>\n"
	for state.currPos = 0; state.currPos < len(state.inpStr); state.currPos++ {
    isNewLine := false
    isSpace := false
		ch := state.inpStr[state.currPos]
		operation := Tokenize(rune(ch))
		switch operation {
		case TokenHeading:
			parsedToken := ParseHeading(state.inpStr, state.currPos)
			if parsedToken.statusCode != ParseSuccess {
				state.printParseError(parsedToken)
				os.Exit(0)
			}
			state.writeBuffer += parsedToken.str
			state.currPos = parsedToken.pos
			state.para.end = true
		case TokenSpace:
			if !state.para.active {
				state.para.begin = true
			}
			if state.isSpace {
				state.writeBuffer += "<br />"
			} else {
				isSpace = true
				state.writeBuffer += " "
			}
		case TokenNewline:
			if !state.para.active {
				// this is for when a newline is written and we want to check if a new paragraph must begin
				state.para.begin = true
			}
			if state.isNewLine {
				// we expect that by the second newline we are already in a paragraph. that is the correct behavior
				// normally, I would have an assertion here to check that para.active is true but this would suffice
				// the rest I shall catch through tests
				state.para.end = true
			} else {
				isNewLine = true
				state.writeBuffer += "\n"
			}
		default:
			state.writeBuffer += string(ch)
			if !state.para.active {
				state.para.begin = true
			}
		}
		state.writeToOutputStr()
		state.writeBuffer = ""
    state.isSpace = isSpace
    state.isNewLine = isNewLine
	}
	// incase something was being parsed as we reached end of string
	// we will attempt to flush the write buffer to the output string
  if state.para.active {
    state.para.end = true
    state.writeToOutputStr()
  }
	state.outStr += "\n</article>"

	return state.outStr
}

func process(src_path string, dst_path string) {
	var state pathState = pathState{
		src_path:  src_path,
		src_files: make([]string, 0, 8),
		src_dirs:  make([]string, 0, 8),
		dst_path:  dst_path,
	}

	entries, err := os.ReadDir(state.src_path)
	if err != nil {
		log.Fatal("Failed to read directory:", state.src_path, "Error:", err)
	}
	for _, file := range entries {
		if file.Name()[0] == '.' {
			// ignoring special files
			continue
		}
		if file.IsDir() {
			state.src_dirs = slices.Insert(state.src_dirs, 0, file.Name())
		} else {
			state.src_files = slices.Insert(state.src_files, 0, file.Name())
		}
	}

	// process_files
	for _, fname := range state.src_files {
		fpath := state.src_path + "/" + fname
		file_bytes, err := os.ReadFile(fpath)
		if err != nil {
			log.Fatal("Failed to read file:", fpath, ". Error:", err)
		}
		if strings.Contains(fname, ".md") {
			// process_md_file
			file_conv := ProcessMD(string(file_bytes))
			file_bytes = []byte(file_conv)
			fname_split := strings.Split(fname, ".")
			fname = fname_split[0] + ".html"
		}

		// write_file
		wpath := state.dst_path + "/" + fname
		os.WriteFile(wpath, file_bytes, 0666)
	}

	// read directories
	for _, dirname := range state.src_dirs {
		sub_src_path := state.src_path + "/" + dirname
		sub_dst_path := state.dst_path + "/" + dirname
		err := os.Mkdir(sub_dst_path, 0750)
		if err != nil && !os.IsExist(err) {
			log.Fatal("Failed to make directory:", dirname, ". Error:", err)
		}
		process(sub_src_path, sub_dst_path)
	}
}

func main() {
	srcDirPtr := flag.String("src_dir", "", "path to blog input files")
	dstDirPtr := flag.String("dst_dir", "", "path to blog output files")

	flag.Parse()

	fmt.Println("Source path:", *srcDirPtr)
	fmt.Println("Destination path:", *dstDirPtr)

	process(*srcDirPtr, *dstDirPtr)

	fmt.Println("finished reading root directory")
}
