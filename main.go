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
)

type convState struct {
	eval_type int
	// header stuff
	headerInd  int
	headerOpen bool
	// italic and bold stuff
	italicBoldOpen bool
}

var hMap []string = []string{"h1", "h2", "h3", "h4", "h5"}
var italicBoldMap [][]string = [][]string{{"<i>", "</i>"}, {"<b>", "</b>"}, {"<i><b>", "</b></i>"}}

func process_md_file(file string) string {
	var out_file string
	var state convState = convState{eval_type: MdNone}
	for _, ch := range file {
		if ch == '#' {
			if state.eval_type != MdHeader {
				state.eval_type = MdHeader
				state.headerOpen = false
				state.headerInd = 0
			} else {
				if state.headerInd < len(hMap)-1 {
					state.headerInd++
				}
			}
			continue
		} else if ch == '*' {
			if state.eval_type == MdNone {
				state.eval_type = MdItalic
			} else if state.eval_type >= MdItalic && state.eval_type < MdItalicBold {
				state.eval_type += 1
			}
			continue
		} else if ch == ' ' {
			if state.eval_type == MdHeader && !state.headerOpen {
				out_file += "<" + hMap[state.headerInd] + ">"
				state.headerOpen = true
				state.eval_type = MdHeaderText
				continue
			}
		} else if ch == '\n' {
			if state.eval_type == MdHeaderText {
				out_file += "</" + hMap[state.headerInd] + ">"
				state.eval_type = MdNone
				state.headerOpen = false
				state.headerInd = 0
			}
		} else {
			if state.eval_type >= MdItalic && state.eval_type <= MdItalicBold {
				ind := state.eval_type - MdItalic
				state.eval_type = MdItalic
				if !state.italicBoldOpen {
					out_file += italicBoldMap[ind][0]
					state.italicBoldOpen = true
				} else {
					out_file += italicBoldMap[ind][1]
					state.italicBoldOpen = false
					state.eval_type = MdNone
				}
			}
		}
		out_file += string(ch)
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

	print("finished reading root directory")
}
