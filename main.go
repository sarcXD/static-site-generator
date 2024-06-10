package main

/*
@todo:
- add error validation, so whenever I am doing something undefined, I raise an error informing in detail,
what it is I am doing and why that is undefined behavior. I hate that markdown can be written on a whim
and you don't even know what you did incorrectly
- md conversion
	* link
	* ul
	* ol
	* table
- custom header
- custom components
@improvements:
- handle paragraph closing when file ends, currently im relying on browsers to clean that up automatically
- handle cleaning up double spaces when trying to make a line space, such that only a single line space is created
*/

/*
@progress:
*/

/*
@done:
- header
- paragraphs
- linebreaks
- italic
- bold
- italic bold
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

// this will be a sort of a state machine
// the elements at top have higher priority and
// can contain elements that fall below. As an example
// - A header is singular and no other element will have it
// - When processing lists, they can have all the elements that fall below
// note: this might be slightly confusing because of the MdHeader not containing any other element,
// rest assured, that is the outlier here.
const (
	MdNone   = iota + 0
	MdCustom // custom component
	MdHeader // h1 to h4
	MdHeaderText
	MdPara         // paragraph
	MdElementStart // Start marker for nestable elements
	MdUl           // unordered list
	MdOl           // ordered list
	MdItalic       // italic
	MdBold         // bold
	MdItalicBold   // italic bold
	MdLink         // link
	MdText         // Normal text
	MdElementEnd   // End marker for nestable elements
	// ------ past this range we have the text buffers, where should the state be that, we will clear the respective buffer
	// and write it's data to the output file
	MdBufferFlushHeader
	MdBufferFlushItalicBold
)

const (
	IBNone = iota + 0
	IBStart
	IBWriting
	IBEnd
	IBFinish
)

type State interface {
	flush_tag()
}

type convState struct {
	eval_type int
	lineBegin bool
	skipWrite bool
	// header stuff
	headerInd    int
	hTokenBuffer string
	// paragraph stuff
	paraBegin    bool
	paraEnd      bool
	newlineCount int
	// linebreak stuff
	isLineBreak bool
	spaceCount  int
	// italic and bold stuff
	ibIndStart    int
	ibIndEnd      int
	ibEval        int
	ibTokenBuffer string
	// parsing tracking
	posline int
	poscol  int
}

var hMap []string = []string{"h1", "h2", "h3", "h4", "h5"}
var paraMap []string = []string{"<p>", "</p>"}
var italicBoldMap [][]string = [][]string{{"<i>", "</i>"}, {"<b>", "</b>"}, {"<i><b>", "</b></i>"}}

func (state convState) flush_tag(flushType int) {
	state.eval_type = flushType
}

func process_md_file(file string) string {
	var out_file string
	var state convState = convState{eval_type: MdNone, posline: 0, poscol: 0}
	state.lineBegin = true
	for _, ch := range file {
		wroteText := false
		ignoreWrite := false
		isLineBegin := false
		state.skipWrite = false
		switch ch {
		case '#':
			// If the line has just begun, we will process the hash character
			if state.lineBegin {
				if state.eval_type != MdHeader {
					// if we are starting header, prepare state variables
					state.eval_type = MdHeader
					state.headerInd = 0
					state.hTokenBuffer = ""
				} else if state.headerInd < len(hMap)-1 {
					// else increment the header index index
					state.headerInd++
				} else {
					// incase we exceed the header index, that is no longer a valid header
					// reset header state in that case
					state.eval_type = MdBufferFlushHeader
				}
			}
			// we'll let it fall through to the hTokenBuffer
			// this will help us deal with the cases where it is supposed to be treated as normal text
			state.hTokenBuffer += string(ch)
			ignoreWrite = true
		case '*':
			switch state.eval_type {
			case MdNone:
				state.eval_type = MdItalic
				state.ibTokenBuffer += string(ch)
				state.ibIndStart = 0
				state.ibEval = IBStart
			case MdItalic:
				if state.ibEval == IBStart {
					state.eval_type = MdBold
					state.ibIndStart += 1
				} else if state.ibEval == IBWriting {
					state.ibEval = IBFinish
					state.ibIndEnd = 0
				} else {
					state.eval_type = MdBufferFlushItalicBold
				}
				state.ibTokenBuffer += string(ch)
			case MdBold:
				if state.ibEval == IBStart {
					state.eval_type = MdItalicBold
					state.ibIndStart += 1
				} else if state.ibEval == IBWriting {
					state.ibEval = IBEnd
					state.ibIndEnd = 0
				} else if state.ibEval == IBEnd {
					state.ibIndEnd += 1
					state.ibEval = IBFinish
				} else {
					state.eval_type = MdBufferFlushItalicBold
				}
				state.ibTokenBuffer += string(ch)
			case MdItalicBold:
				if state.ibEval == IBWriting {
					state.ibEval = IBEnd
					state.ibIndEnd = 0
				} else if state.ibEval == IBEnd {
					state.ibIndEnd += 1
				} else {
					state.eval_type = MdBufferFlushItalicBold
				}

				if state.ibIndEnd == 2 {
					state.ibEval = IBFinish
				} else if state.ibIndEnd > 2 {
					// this is an error case (****)
					state.eval_type = MdBufferFlushItalicBold
				}
			default:
				// do nothing
			}
			ignoreWrite = true
		case ' ':
			switch state.eval_type {
			case MdHeader:
				out_file += "<" + hMap[state.headerInd] + ">"
				state.eval_type = MdHeaderText
				ignoreWrite = true
			case MdItalic, MdBold, MdItalicBold:
				if state.ibEval == IBStart || state.ibEval == IBEnd {
					state.eval_type = MdBufferFlushItalicBold
				}
			default:
				state.spaceCount += 1
			}
		case '\n':
			switch state.eval_type {
			case MdHeaderText:
				out_file += "</" + hMap[state.headerInd] + ">"
				state.eval_type = MdNone
				state.headerInd = 0
				state.paraBegin = true
			default:
				if state.spaceCount == 2 {
					state.isLineBreak = true
					state.spaceCount = 0
				}
				state.newlineCount += 1
			}
			if state.newlineCount == 2 {
				state.paraEnd = true
			}
			isLineBegin = true
		default:
			// in case we have to write normal text handle each case accordingly
			switch state.eval_type {
			case MdHeader:
				// incase a header is active, normal text is not valid before a ' ' character.
				// the valid state would be MdHeaderBuffer
				// in this case we will set the state to MdBufferFlushHeader
				state.eval_type = MdBufferFlushHeader
			case MdItalic, MdBold, MdItalicBold:
				if state.ibEval == IBStart {
					out_file += italicBoldMap[state.ibIndStart][0]
					state.ibEval = IBWriting
					state.ibTokenBuffer = ""
				}
			default:
				wroteText = true
			}
		}
		// Explicit Writing
		// In certain cases, the state is set, and now we need to write the relevant tags
		// This section handles that
		if (state.eval_type > MdNone &&
			state.eval_type < MdElementEnd) || wroteText {
			state.spaceCount = 0
			state.newlineCount = 0
		}
		// --- linebreak
		if state.isLineBreak {
			out_file += "<br>"
			state.isLineBreak = false
		}
		// --- paragraph
		if state.paraBegin {
			out_file += paraMap[0]
			state.paraBegin = false
		}
		if state.paraEnd {
			out_file += paraMap[1]
			state.paraEnd = false
			state.paraBegin = true
		}
		// --- italic, bold, italic bold
		if state.eval_type == MdItalic ||
			state.eval_type == MdBold ||
			state.eval_type == MdItalicBold {
			if state.ibEval == IBFinish {
				out_file += italicBoldMap[state.ibIndEnd][1]
				state.ibEval = IBNone
				state.eval_type = MdNone
				state.ibTokenBuffer = ""
			}
		}
		// flush_token_buffer
		// check if we have any buffer state
		// buffer states mean that the buffer needs to be written to output file and cleared
		// these states occur whenever some markdown we were parsing turns out to be invalid
		if state.eval_type >= MdBufferFlushHeader {
			switch state.eval_type {
			case MdBufferFlushHeader:
				out_file += state.hTokenBuffer
				state.hTokenBuffer = ""
				log.Printf("ParsingError :: Invalid header format at line %d, col %d", state.posline, state.poscol)
			case MdBufferFlushItalicBold:
				out_file += state.ibTokenBuffer
				state.ibTokenBuffer = ""
				state.ibIndStart = 0
				state.ibEval = IBNone
				log.Printf("ParsingError :: Invalid italic/bold/italic bold format at line %d, col %d", state.posline, state.poscol)
			default:
				log.Printf("Warning, state.eval_type value = %d, has no buffer flush\n", state.eval_type)
			}
			state.eval_type = MdNone
		}
		state.lineBegin = isLineBegin
		if !ignoreWrite {
			out_file += string(ch)
		}
		if state.lineBegin {
			state.posline += 1
			state.poscol = 0
		}
	}

	return out_file
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
			file_conv := process_md_file(string(file_bytes))
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
