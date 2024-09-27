package utils

import (
	"fmt"
	ct "github.com/daviddengcn/go-colortext"
	"io"
	"os"
)

func PrintYellow(out io.Writer, content string) {
	ct.ChangeColor(ct.Yellow, false, ct.None, false)
	_, _ = fmt.Fprint(out, content)
	ct.ResetColor()
}

func PrintWarning(out io.Writer, name string) {
	ct.ChangeColor(ct.Red, false, ct.None, false)
	_, _ = fmt.Fprint(out, name)
	ct.ResetColor()
}

func PrintString(out io.Writer, name string) {
	ct.ChangeColor(ct.Green, false, ct.None, false)
	_, _ = fmt.Fprint(out, name)
	ct.ResetColor()
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func SplitCSKV(s string) (map[string]string, error) {
	state := "key"
	key := ""
	val := ""
	result := map[string]string{}
	procKV := func() {
		if key != "" {
			result[key] = val
		}
		state = "key"
		key = ""
		val = ""
	}
	for _, c := range s {
		switch state {
		case "key":
			switch c {
			case '"':
				state = "keyQuote"
			case '\\':
				state = "keyEscape"
			case '=':
				state = "val"
			case ',':
				procKV()
			default:
				key = key + string(c)
			}
		case "keyQuote":
			switch c {
			case '"':
				state = "key"
			case '\\':
				state = "keyEscapeQuote"
			default:
				key = key + string(c)
			}
		case "keyEscape":
			key = key + string(c)
			state = "key"
		case "keyEscapeQuote":
			key = key + string(c)
			state = "keyQuote"
		case "val":
			switch c {
			case '"':
				state = "valQuote"
			case ',':
				procKV()
			case '\\':
				state = "valEscape"
			default:
				val = val + string(c)
			}
		case "valQuote":
			switch c {
			case '"':
				state = "val"
			case '\\':
				state = "valEscapeQuote"
			default:
				val = val + string(c)
			}
		case "valEscape":
			val = val + string(c)
			state = "val"
		case "valEscapeQuote":
			val = val + string(c)
			state = "valQuote"
		default:
			return nil, fmt.Errorf("unhandled state: %s", state)
		}
	}
	switch state {
	case "val", "key":
		procKV()
	default:
		return nil, fmt.Errorf("string parsing failed, end state: %s", state)
	}
	return result, nil
}
