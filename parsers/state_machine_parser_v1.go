package state_machine_parser_v1;

// this will be a sort of a state machine
// the elements at top have higher priority and
// can contain elements that fall below. As an example
// - A header is singular and no other element will have it
// - When processing lists, they can have all the elements that fall below
// note: this might be slightly confusing because of the MdHeader not containing any other element,
// rest assured, that is the outlier here.
const (
	MdNone    = iota + 0
	MdCustom  // custom component
	MdHeader  // h1 to h4
	MdElement // normal elements
	// @note:
	// ------ past this range we have the text buffer states
	// MdFlushWrite: flush data to output file
	// MdFlushError: handle error and flush the output from that (raw text) into the rawBuffer
	// -----------------------------------------------------------
	// SUBJECT TO CHANGE
	// MdBufferFlush${OperatorName}: this error type points to error in a particular type of formatter
	// and we need to handle each case usually differently.
	MdFlushWrite
	MdFlushError
	MdBufferFlushHeader
	MdBufferFlushItalicBold
	MdDropWrite
)

// header parsing state
const (
	hStart = iota + 0
	hText
)

// italic bold type
const (
	ibNone = iota + 0
	ibItalic
	ibBold
	ibItalicBold
)

// italic bold parsing state
const (
	ibEleNone = iota + 0
	ibEleStart
	ibEleWriting
	ibEleEnd
	ibEleFinish
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

	ibEval         int
	ibIndexStart   int
	ibIndexEnd     int
	ibElementEval  int
	ibBufferRaw    string
	ibBufferParsed string

	// parsing tracking
	posline int
	poscol  int
}

var hMap []string = []string{"h1", "h2", "h3", "h4", "h5", "h6"}
var paraMap []string = []string{"<p>", "</p>"}
var italicBoldMap [][]string = [][]string{{"<i>", "</i>"}, {"<b>", "</b>"}, {"<i><b>", "</b></i>"}}

func ProcessMdFileSMv0(file string) string {
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
		case "*":
			if !state.paraActive {
				state.paraBegin = true
			}
			if state.stateEval == MdNone || state.stateEval == MdElement {
				if state.ibEval == ibNone {
					state.stateEval = MdElement
					state.ibEval = ibItalic
					state.ibElementEval = ibEleStart
					state.ibBufferRaw = ""
					state.ibBufferParsed = ""
					state.ibIndexStart = 1
					state.ibIndexEnd = 1
				} else if state.ibEval > ibNone {
					if state.ibElementEval == ibEleStart {
						if state.ibEval == ibItalic {
							// make state bold
							state.ibEval = ibBold
							state.ibIndexStart++
							state.ibIndexEnd++
						} else if state.ibEval == ibBold {
							// make state italic bold
							state.ibEval = ibItalicBold
							state.ibIndexStart++
							state.ibIndexEnd++
						} else {
							// error case
							state.stateEval = MdBufferFlushItalicBold
						}
					} else if state.ibElementEval > ibEleStart {
						if state.ibEval == ibItalic {
							state.ibElementEval = ibEleEnd
							state.ibIndexEnd--
						} else if state.ibEval == ibBold {
							state.ibElementEval = ibEleEnd
							state.ibIndexEnd--
						} else if state.ibEval == ibItalicBold {
							state.ibElementEval = ibEleEnd
							state.ibIndexEnd--
						} else {
							state.stateEval = MdBufferFlushItalicBold
						}
						if state.ibIndexEnd == 0 {
							state.ibEval = ibNone
							state.ibElementEval = ibEleNone
							state.ibBufferParsed += italicBoldMap[state.ibIndexStart-1][1]
							state.writeBuffer += state.ibBufferParsed
							state.stateEval = MdFlushWrite
						}
					}
				}
				state.ibBufferRaw += ch
			} else {
				state.writeBuffer += ch
			}
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
				}
				if state.stateEval == MdElement {
					if state.ibEval > ibNone {
						if state.ibElementEval != ibEleWriting {
							state.stateEval = MdBufferFlushItalicBold
						}
						state.ibBufferRaw += ch
						state.ibBufferParsed += ch
					}
				} else {
					if !state.isPageBreak {
						state.writeBuffer += ch
						state.stateEval = MdFlushWrite
					}
				}
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
			} else if state.ibEval > ibNone {
				state.ibBufferRaw += ch
				state.ibBufferParsed += ch
			} else {
				if state.lineBegin {
					// we have a double line.
					// close the paragraph
					if state.paraActive {
						state.paraEnd = true
						state.stateEval = MdDropWrite
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
			} else if state.stateEval == MdElement {
				if state.ibEval > MdNone && state.ibElementEval == ibEleStart {
					state.ibElementEval = ibEleWriting
					state.ibBufferParsed += italicBoldMap[state.ibIndexStart-1][0]
				}
				state.ibBufferRaw += ch
				state.ibBufferParsed += ch
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
			case MdBufferFlushItalicBold:
				state.writeBuffer += state.ibBufferRaw
				state.stateEval = MdFlushWrite
				state.ibEval = ibNone
				state.ibElementEval = ibEleNone
				log.Printf("Warning::Incorrect ib format at line %d, col %d", state.posline, state.poscol)
			}
		}
		// drop write buffer
		if state.stateEval == MdDropWrite {
			state.writeBuffer = ""
			state.stateEval = MdNone
		}
		// prefix write
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
		if state.isPageBreak {
			// postfix write
			out_file += "<br />"
			state.isPageBreak = false
		}

		state.lineBegin = lineBegin
		state.isSpace = isSpace
		if state.lineBegin {
			state.posline += 1
			state.poscol = 0
		}
	}
	// @note: We need to handle the cases where some formatting had begun and
	// the file string ended before those formats were properly closed.
	// these are all errors and to handle them, the respective raw strings need to
	// be flushed to the output file
	if state.paraBegin {
		out_file += "\n<p>"
		state.paraActive = true
	}
	if state.stateEval == MdElement {
		if state.ibEval != ibNone {
			out_file += state.ibBufferRaw
		}
	}
	if state.paraActive {
		out_file += "</p>\n"
		state.paraActive = false
	}
	out_file += "\n</article>"

	return out_file
}
