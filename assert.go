package assert

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

func isNil(val interface{}) (ret bool) {
	defer func() {
		if e := recover(); e != nil {
			ret = false
		}
	}()

	if val == nil {
		return true
	}

	return reflect.ValueOf(val).IsNil()
}

func getFileLine(skip uint) string {
	sb := &bytes.Buffer{}

	if _, file, line, ok := runtime.Caller(int(skip) + 1); ok && line > 0 {
		sb.WriteString(file)
		sb.WriteByte(':')
		sb.WriteString(strconv.Itoa(line))
	}

	return sb.String()
}

func addPrefixPerLine(text string, prefix string) string {
	sb := &bytes.Buffer{}

	first := true
	array := strings.Split(text, "\n")
	for idx, v := range array {
		if first {
			first = false
		} else {
			sb.WriteByte('\n')
		}

		if v != "" || idx == 0 || idx != len(array)-1 {
			sb.WriteString(prefix)
			sb.WriteString(v)
		}
	}
	return sb.String()
}

func convertOrdinalToString(n uint) string {
	if n == 0 {
		return ""
	}

	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return strconv.Itoa(int(n)) + "th"
	}
}

// Assert ...
type Assert struct {
	t    interface{ Fail() }
	args []interface{}
}

// New create a new assert function
func New(t interface{ Fail() }) func(args ...interface{}) *Assert {
	return func(args ...interface{}) *Assert {
		return &Assert{
			t:    t,
			args: args,
		}
	}
}

func (p *Assert) fail(reason string) {
	_, _ = os.Stdout.WriteString(
		fmt.Sprintf("\t%s\n\t%s\n", reason, getFileLine(2)),
	)
	p.t.Fail()
}

// Fail report test failure
func (p *Assert) Fail(reason string) {
	p.fail(reason)
}

// Equals if all parameters are equal. Report that the test was successful,
// Otherwise report test failure
func (p *Assert) Equals(args ...interface{}) {
	if len(p.args) < 1 {
		p.fail("arguments is empty")
	} else if len(p.args) != len(args) {
		p.fail("arguments length not match")
	} else {
		for i := 0; i < len(p.args); i++ {
			if !reflect.DeepEqual(p.args[i], args[i]) {
				if !isNil(p.args[i]) || !isNil(args[i]) {
					p.fail(fmt.Sprintf(
						"%s argument does not equal\n\twant:\n%s\n\tgot:\n%s",
						convertOrdinalToString(uint(i+1)),
						addPrefixPerLine(fmt.Sprintf(
							"%T(%v)", args[i], args[i]), "\t",
						),
						addPrefixPerLine(fmt.Sprintf(
							"%T(%v)", p.args[i], p.args[i]), "\t",
						),
					))
				}
			}
		}
	}
}

// IsNil if all parameters are nil. Report that the test was successful,
// Otherwise report test failure
func (p *Assert) IsNil() {
	if len(p.args) < 1 {
		p.fail("arguments is empty")
	} else {
		for i := 0; i < len(p.args); i++ {
			if !isNil(p.args[i]) {
				p.fail(fmt.Sprintf(
					"%s argument is not nil",
					convertOrdinalToString(uint(i+1)),
				))
			}
		}
	}
}

// IsNotNil if all parameters are not nil. Report that the test was successful,
// Otherwise report test failure
func (p *Assert) IsNotNil() {
	if len(p.args) < 1 {
		p.fail("arguments is empty")
	} else {
		for i := 0; i < len(p.args); i++ {
			if isNil(p.args[i]) {
				p.fail(fmt.Sprintf(
					"%s argument is nil",
					convertOrdinalToString(uint(i+1)),
				))
			}
		}
	}
}

// IsTrue if all parameters are true. Report that the test was successful,
// Otherwise report test failure
func (p *Assert) IsTrue() {
	if len(p.args) < 1 {
		p.fail("arguments is empty")
	} else {
		for i := 0; i < len(p.args); i++ {
			if p.args[i] != true {
				p.fail(fmt.Sprintf(
					"%s argument is not true",
					convertOrdinalToString(uint(i+1)),
				))
			}
		}
	}
}

// IsFalse if all parameters are false. Report that the test was successful,
// Otherwise report test failure
func (p *Assert) IsFalse() {
	if len(p.args) < 1 {
		p.fail("arguments is empty")
	} else {
		for i := 0; i < len(p.args); i++ {
			if p.args[i] != false {
				p.fail(fmt.Sprintf(
					"%s argument is not false",
					convertOrdinalToString(uint(i+1)),
				))
			}
		}
	}
}
