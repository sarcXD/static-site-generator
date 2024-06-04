package main

/*
@todo:
- add error validation, so whenever I am doing something undefined, I raise an error informing in detail,
what it is I am doing and why that is undefined behavior. I hate that markdown can be written on a whim
and you don't even know what you did incorrectly
- md conversion
	* italic
	* bold
	* italic bold
	* link
	* ul
	* ol
	* table
- custom header
- custom components
*/

/*
@resume:
- working on italic and bold formattings. That is completely broken right now
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
const (
	MdNone   = iota + 0
	MdHeader // h1 to h4
	MdHeaderText
	MdItalic     // italic
	MdBold       // bold
	MdItalicBold // italic bold
	MdLink       // link
	MdUl         // unordered list
	MdOl         // ordered list
	MdCustom     // custom component
	// ------ past this range we have the text buffers, where should the state be that, we will clear the respective buffer
	// and write it's data to the output file
	MdBufferFlushHeader
)

const (
	IBNone = iota + 0
	IBStart
	IBStartDone
	IBEnd
	IBEndDone
)

type convState struct {
	eval_type int
	lineBegin bool
	skipWrite bool
	// header stuff
	headerInd    int
	hTokenBuffer string
	// italic and bold stuff
	ib_eval        int
	italicBoldOpen bool
	ibTokenBuffer  string
}

var hMap []string = []string{"h1", "h2", "h3", "h4", "h5"}
var italicBoldMap [][]string = [][]string{{"<i>", "</i>"}, {"<b>", "</b>"}, {"<i><b>", "</b></i>"}}

func process_md_file(file string) string {
	var out_file string
	var state convState = convState{eval_type: MdNone, ib_eval: IBNone}
	state.lineBegin = true
	for _, ch := range file {
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
			if false {
				state.ibTokenBuffer += string(ch)
				if state.ib_eval == IBNone {
					state.ib_eval = IBStart
				} else if state.ib_eval == IBStartDone {
					state.ib_eval = IBEnd
				}

				if state.eval_type == MdNone {
					state.eval_type = MdItalic
				} else if state.eval_type >= MdItalic && state.eval_type < MdItalicBold {
					state.eval_type += 1
				}
				state.skipWrite = true
			}
		case ' ':
			switch state.eval_type {
			case MdHeader:
				out_file += "<" + hMap[state.headerInd] + ">"
				state.eval_type = MdHeaderText
				ignoreWrite = true
			case MdNone:
			default:
				fmt.Printf("Warning, state.eval_type value = %d, has no handling for space operator/character \n", state.eval_type)
			}
			if false {
				if state.ib_eval == IBStart {
					// ** xyz.. |=> this is not valid as we need the characters right besides the asterisk for a valid italic bold command
					out_file += state.ibTokenBuffer
					state.ib_eval = IBNone
				}
			}
		case '\n':
			switch state.eval_type {
			case MdHeaderText:
				out_file += "</" + hMap[state.headerInd] + ">"
				state.eval_type = MdNone
				state.headerInd = 0
			default:
				// do nothing
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
			default:
				fmt.Printf("Warning, state.eval_type value = %d, has no handling for new line operator/character \n", state.eval_type)
			}
			if false {
				if state.eval_type >= MdItalic && state.eval_type <= MdItalicBold {
					ind := state.eval_type - MdItalic
					if !state.italicBoldOpen {
						out_file += italicBoldMap[ind][0]
						state.italicBoldOpen = true
						state.ib_eval = IBStartDone
					} else {
						out_file += italicBoldMap[ind][1]
						state.italicBoldOpen = false
						state.ib_eval = IBEndDone
					}
					state.eval_type = MdNone
					state.ibTokenBuffer = ""
				}
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
			default:
				log.Printf("Warning, state.eval_type value = %d, has no buffer flush\n", state.eval_type)
			}
			state.eval_type = MdNone
		}
		state.lineBegin = isLineBegin
		if !ignoreWrite {
			out_file += string(ch)
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

	println("finished reading root directory")
}
