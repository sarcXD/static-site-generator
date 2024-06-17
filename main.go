package main

/*
@important:
- add error validation, so whenever I am doing something undefined, I raise an error informing in detail,
what it is I am doing and why that is undefined behavior. I hate that markdown can be written on a whim
and you don't even know what you did incorrectly

@todo:
---
- text formatting
-- italic
-- bold
-- italicBold
- md conversion
	* link
	* ul
	* ol
- table? (probably a custom table)
- custom header
- custom components

@improvements:
@progress:
- text formatting
@done:
- headings
- paragraphs
- linebreak
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
	MdPara
	MdElement // Start marker for nestable elements
	// ------ past this range we have the text buffers, where should the state be that, we will clear the respective buffer
	// and write it's data to the output file
	MdFlushWrite
	MdFlushError
	MdBufferFlushHeader
	MdBufferFlushItalicBold
)

const (
	wMDNone = iota + 0
	wMDHeader
)

// header parsing state
const (
	hStart = iota + 0
	hText
	hWrite
)

// italic bold type
const (
	IBItalic = iota + 0
	IBBold
	IBItalicBold
)

// italic bold parsing state
const (
	ibNone = iota + 0
	ibStart
	ibWriting
	ibEnd
	ibFinish
)

// link parsing state
const (
	linkNone = iota + 0
	linkStart
	linkText
	linkUrl
	linkFinish
)

// maybe?
// @note: (rawBuffer)
// this buffer writes text as if it was just normal text. This is so that when the tag I am parsing is incorrect,
// I have the raw data available so I can write it as is.

type convState struct {
	stateEval int
	lineBegin bool
	// == header stuff ==
	headerBufferRaw    string // if parse error: flush this buffer to out_file
	headerBufferParsed string // if ok: flush this buffer to out file
	headerIndex        int
	headerEval         int

	writeBuffer string

	paraBegin    bool
	paraEnd      bool
	paraSurround bool
	paraActive   bool

	isSpace     bool
	isPageBreak bool
	// parsing tracking
	posline int
	poscol  int
}

var hMap []string = []string{"h1", "h2", "h3", "h4", "h5", "h6"}
var paraMap []string = []string{"<p>", "</p>"}
var italicBoldMap [][]string = [][]string{{"<i>", "</i>"}, {"<b>", "</b>"}, {"<i><b>", "</b></i>"}}

func process_md_file(file string) string {
	var out_file string
	var state convState = convState{stateEval: MdNone, posline: 0, poscol: 0}
  out_file = "<article>\n"
	state.lineBegin = true
	for i := 0; i < len(file); i++ {
		isSpace := false
		lineBegin := false
		ch := string(file[i])
		switch ch {
		case "#":
			if state.stateEval == MdNone && state.lineBegin {
				state.stateEval = MdHeader
				state.headerBufferParsed = ""
				state.headerBufferRaw = ""
				state.headerIndex = 0
				state.headerEval = hStart
			} else if state.stateEval == MdHeader {
				if state.headerEval == hStart {
					if state.headerIndex >= len(hMap)-1 {
						state.stateEval = MdBufferFlushHeader
					} else {
						state.headerIndex += 1
					}
				} else {
					state.headerBufferParsed += ch
				}
			}
			state.headerBufferRaw += ch
		case " ":
      isSpace = true
			if state.stateEval == MdHeader {
				if state.headerEval == hStart {
					state.headerEval = hText
					state.headerBufferParsed += "<" + hMap[state.headerIndex] + ">"
				} else {
					state.headerBufferParsed += ch
				}
				state.headerBufferRaw += ch
			} else {
				if state.isSpace {
					state.isPageBreak = true
				} else {
					state.writeBuffer += ch
				}
				state.stateEval = MdFlushWrite
			}
		case "\n":
			lineBegin = true
			if state.stateEval == MdHeader {
				if state.headerEval == hStart {
					state.headerEval = MdBufferFlushHeader
				} else {
					state.headerBufferParsed += "</" + hMap[state.headerIndex] + ">"
					state.headerBufferParsed += ch
					state.stateEval = MdFlushWrite
					state.writeBuffer += state.headerBufferParsed

					if state.paraActive {
						state.paraEnd = true
					}
				}
				state.headerBufferRaw += ch
			} else {
				if state.lineBegin {
					// we have a double line.
					// close the paragraph
					if state.paraActive {
						state.paraEnd = true
					}
				} else {
					state.writeBuffer += ch
				}
			}
		default:
			if state.stateEval == MdHeader {
				if state.headerEval == hStart {
					state.stateEval = MdBufferFlushHeader
				} else {
					state.headerBufferParsed += ch
				}
				state.headerBufferRaw += ch
			}
			if state.stateEval == MdNone {
				if !state.paraActive {
					state.paraBegin = true
				}
				state.writeBuffer += ch
				state.stateEval = MdFlushWrite
			}
		}
		// Error checking is done first,
		// any data to flush is sent to write buffer
		if state.stateEval > MdFlushError {
			switch state.stateEval {
			case MdBufferFlushHeader:
				state.writeBuffer += state.headerBufferRaw
				state.stateEval = MdFlushWrite
				if !state.paraActive {
					state.paraBegin = true
				}
				log.Printf("Warning::Incorrect header at line %d, col %d", state.posline, state.poscol)
			}
		}

    if state.isPageBreak {
      out_file += "<br />"
      state.isPageBreak = false
    }
		if state.paraEnd {
			out_file += "</p>\n"
			state.paraEnd = false
			state.paraActive = false
		}
		// Check to see if any data needs flushing
		if state.stateEval == MdFlushWrite {
			if state.paraBegin {
				out_file += "\n<p>"
				state.paraBegin = false
				state.paraActive = true
			}
			out_file += state.writeBuffer
			if state.paraSurround {
				out_file += "</p>\n"
				state.paraSurround = false
				state.paraActive = false
			}
			state.stateEval = MdNone
			state.writeBuffer = ""
		}

		state.lineBegin = lineBegin
		state.isSpace = isSpace
		if state.lineBegin {
			state.posline += 1
			state.poscol = 0
		}
	}
	// some token closing checks need to be repeated
	// in case the file ends without a newline/double newline
	if state.paraActive {
		out_file += "</p>\n"
		state.paraActive = false
	}
  out_file += "\n</article>"

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
