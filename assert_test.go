package assert

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"unsafe"
)

func captureStdout(fn func()) string {
	oldStdout := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	func() {
		defer func() {
			_ = recover()
		}()
		fn()
	}()

	outCH := make(chan string)
	// copy the output in a separate goroutine so print can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outCH <- buf.String()
	}()

	os.Stdout = oldStdout
	_ = w.Close()
	ret := <-outCH
	_ = r.Close()
	return ret
}

type fakeTesting struct {
	onFail func()
}

func (p *fakeTesting) Fail() {
	if p.onFail != nil {
		p.onFail()
	}
}

func testFailHelper(fn func(_ func(_ ...interface{}) *Assert)) (bool, string) {
	retCH := make(chan bool, 1)

	retValue := captureStdout(func() {
		fn(func(args ...interface{}) *Assert {
			return &Assert{
				t: &fakeTesting{
					onFail: func() {
						retCH <- true
					},
				},
				args: args,
			}
		})
	})

	select {
	case <-retCH:
		return true, retValue
	default:
		return false, retValue
	}
}

func TestIsNil(t *testing.T) {
	t.Run("test", func(t *testing.T) {
		assert := New(t)
		assert(isNil(nil)).IsTrue()
		assert(isNil(t)).IsFalse()
		assert(isNil(3)).IsFalse()
		assert(isNil(0)).IsFalse()
		assert(isNil(uintptr(0))).IsFalse()
		assert(isNil(uintptr(1))).IsFalse()
		assert(isNil(unsafe.Pointer(nil))).IsTrue()
		assert(isNil(unsafe.Pointer(t))).IsFalse()
	})
}

func TestGetFileLine(t *testing.T) {
	t.Run("test", func(t *testing.T) {
		assert := New(t)
		fileLine1 := getFileLine(0)
		assert(strings.Contains(fileLine1, "assert_test.go:")).IsTrue()
	})
}

func TestAddPrefixPerLine(t *testing.T) {
	t.Run("test", func(t *testing.T) {
		assert := New(t)
		assert(addPrefixPerLine("", "")).Equals("")
		assert(addPrefixPerLine("a", "")).Equals("a")
		assert(addPrefixPerLine("\n", "")).Equals("\n")
		assert(addPrefixPerLine("a\n", "")).Equals("a\n")
		assert(addPrefixPerLine("a\nb", "")).Equals("a\nb")
		assert(addPrefixPerLine("", "-")).Equals("-")
		assert(addPrefixPerLine("a", "-")).Equals("-a")
		assert(addPrefixPerLine("\n", "-")).Equals("-\n")
		assert(addPrefixPerLine("a\n", "-")).Equals("-a\n")
		assert(addPrefixPerLine("a\nb", "-")).Equals("-a\n-b")
	})
}

func TestConvertOrdinalToString(t *testing.T) {
	t.Run("test", func(t *testing.T) {
		assert := NewAssert(t)
		assert(ConvertOrdinalToString(0)).Equals("")
		assert(ConvertOrdinalToString(1)).Equals("1st")
		assert(ConvertOrdinalToString(2)).Equals("2nd")
		assert(ConvertOrdinalToString(3)).Equals("3rd")
		assert(ConvertOrdinalToString(4)).Equals("4th")
		assert(ConvertOrdinalToString(10)).Equals("10th")
		assert(ConvertOrdinalToString(100)).Equals("100th")
	})
}

func TestNewAssert(t *testing.T) {
	t.Run("t is nil", func(t *testing.T) {
		assert := New(t)
		o := New(nil)
		assert(o(true)).Equals(&Assert{t: nil, args: []interface{}{true}})
	})

	t.Run("args is nil", func(t *testing.T) {
		assert := New(t)
		o := New(t)
		assert(o()).Equals(&Assert{t: t, args: nil})
	})

	t.Run("test", func(t *testing.T) {
		assert := New(t)
		o := New(t)
		assert(o(true, 1)).Equals(&Assert{t: t, args: []interface{}{true, 1}})
	})
}

func TestRpcAssert_Fail(t *testing.T) {
	t.Run("test", func(t *testing.T) {
		assert := New(t)
		source := ""
		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o().Fail("error"); source = getFileLine(0) }()
		})).Equals(true, "\terror\n\t"+source+"\n")
	})
}

func TestRpcAssert_Equals(t *testing.T) {
	t.Run("arguments is empty", func(t *testing.T) {
		assert := New(t)
		source := ""
		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o().Equals(); source = getFileLine(0) }()
		})).Equals(true, "\targuments is empty\n\t"+source+"\n")
	})

	t.Run("arguments is empty", func(t *testing.T) {
		assert := New(t)
		source := ""
		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(1).Equals(1, 2); source = getFileLine(0) }()
		})).Equals(true, "\targuments length not match\n\t"+source+"\n")
	})

	t.Run("arguments does not equal", func(t *testing.T) {
		assert := New(t)
		source := ""
		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(1).Equals(2); source = getFileLine(0) }()
		})).Equals(true, fmt.Sprintf(
			"\t1st argument does not equal\n\t"+
				"want:\n\t%s\n\tgot:\n\t%s\n\t%s\n",
			"int(2)",
			"int(1)",
			source,
		))
	})

	t.Run("arguments type does not equal", func(t *testing.T) {
		assert := New(t)
		source := ""

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(1).Equals(int64(1)); source = getFileLine(0) }()
		})).Equals(true, fmt.Sprintf(
			"\t1st argument does not equal\n\t"+
				"want:\n\t%s\n\tgot:\n\t%s\n\t%s\n",
			"int64(1)",
			"int(1)",
			source,
		))

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			v1 := map[int]interface{}{3: "OK", 4: []byte(nil)}
			v2 := map[int]interface{}{3: "OK", 4: nil}
			func() { o(v1).Equals(v2); source = getFileLine(0) }()
		})).Equals(true, fmt.Sprintf(
			"\t1st argument does not equal\n\t"+
				"want:\n\t%s\n\tgot:\n\t%s\n\t%s\n",
			"map[int]interface {}(map[3:OK 4:<nil>])",
			"map[int]interface {}(map[3:OK 4:[]])",
			source,
		))

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			v1 := []int{1, 2, 3}
			v2 := []int64{1, 2, 3}
			func() { o(v1).Equals(v2); source = getFileLine(0) }()
		})).Equals(true, fmt.Sprintf(
			"\t1st argument does not equal\n\t"+
				"want:\n\t%s\n\tgot:\n\t%s\n\t%s\n",
			"[]int64([1 2 3])",
			"[]int([1 2 3])",
			source,
		))

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			v1 := []int{1, 2, 3}
			v2 := []int{1, 3, 2}
			func() { o(v1).Equals(v2); source = getFileLine(0) }()
		})).Equals(true, fmt.Sprintf(
			"\t1st argument does not equal\n\t"+
				"want:\n\t%s\n\tgot:\n\t%s\n\t%s\n",
			"[]int([1 3 2])",
			"[]int([1 2 3])",
			source,
		))

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			v1 := make([]interface{}, 0)
			func() { o(v1).Equals(nil); source = getFileLine(0) }()
		})).Equals(true, fmt.Sprintf(
			"\t1st argument does not equal\n\t"+
				"want:\n\t%s\n\tgot:\n\t%s\n\t%s\n",
			"<nil>(<nil>)",
			"[]interface {}([])",
			source,
		))

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			v1 := map[string]interface{}{}
			func() { o(v1).Equals(nil); source = getFileLine(0) }()
		})).Equals(true, fmt.Sprintf(
			"\t1st argument does not equal\n\t"+
				"want:\n\t%s\n\tgot:\n\t%s\n\t%s\n",
			"<nil>(<nil>)",
			"map[string]interface {}(map[])",
			source,
		))
	})

	t.Run("test", func(t *testing.T) {
		assert := New(t)
		assert(3).Equals(3)
		assert(nil).Equals(nil)
		assert((interface{})(nil)).Equals(nil)
		assert([]interface{}(nil)).Equals(nil)
		assert(map[string]interface{}(nil)).Equals(nil)
		assert((*Assert)(nil)).Equals(nil)
		assert(nil).Equals((*Assert)(nil))
		assert(nil).Equals((interface{})(nil))
		assert([]int{1, 2, 3}).Equals([]int{1, 2, 3})
		assert(map[int]string{3: "OK", 4: "NO"}).
			Equals(map[int]string{4: "NO", 3: "OK"})
		assert(1, 2, 3).Equals(1, 2, 3)
	})
}

func TestRpcAssert_IsNil(t *testing.T) {
	t.Run("arguments is empty", func(t *testing.T) {
		assert := New(t)
		source := ""
		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o().IsNil(); source = getFileLine(0) }()
		})).Equals(true, "\targuments is empty\n\t"+source+"\n")
	})

	t.Run("arguments is not nil", func(t *testing.T) {
		assert := New(t)
		source := ""
		getFL := getFileLine

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o([]interface{}{}).IsNil(); source = getFL(0) }()
		})).Equals(true, "\t1st argument is not nil\n\t"+source+"\n")

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(map[string]interface{}{}).IsNil(); source = getFL(0) }()
		})).Equals(true, "\t1st argument is not nil\n\t"+source+"\n")

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(uintptr(0)).IsNil(); source = getFL(0) }()
		})).Equals(true, "\t1st argument is not nil\n\t"+source+"\n")

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(nil, 0).IsNil(); source = getFL(0) }()
		})).Equals(true, "\t2nd argument is not nil\n\t"+source+"\n")
	})

	t.Run("test", func(t *testing.T) {
		assert := New(t)
		assert(nil).IsNil()
		assert(([]interface{})(nil)).IsNil()
		assert((map[string]interface{})(nil)).IsNil()
		assert((interface{})(nil)).IsNil()
		assert((*Assert)(nil)).IsNil()
		assert((unsafe.Pointer)(nil)).IsNil()
		assert(nil, (interface{})(nil)).IsNil()
	})
}

func TestRpcAssert_IsNotNil(t *testing.T) {
	t.Run("arguments is empty", func(t *testing.T) {
		assert := New(t)
		source := ""
		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o().IsNotNil(); source = getFileLine(0) }()
		})).Equals(true, "\targuments is empty\n\t"+source+"\n")
	})

	t.Run("arguments is nil", func(t *testing.T) {
		assert := New(t)
		source := ""

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			getFL := getFileLine
			func() { o([]interface{}(nil)).IsNotNil(); source = getFL(0) }()
		})).Equals(true, "\t1st argument is nil\n\t"+source+"\n")

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			v1 := map[string]interface{}(nil)
			func() { o(v1).IsNotNil(); source = getFileLine(0) }()
		})).Equals(true, "\t1st argument is nil\n\t"+source+"\n")

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(nil).IsNotNil(); source = getFileLine(0) }()
		})).Equals(true, "\t1st argument is nil\n\t"+source+"\n")

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(0, nil).IsNotNil(); source = getFileLine(0) }()
		})).Equals(true, "\t2nd argument is nil\n\t"+source+"\n")
	})

	t.Run("test", func(t *testing.T) {
		assert := New(t)
		assert(0).IsNotNil()
		assert([]interface{}{}).IsNotNil()
		assert(map[string]interface{}{}).IsNotNil()
		assert(uintptr(0)).IsNotNil()
		assert(0, []interface{}{}).IsNotNil()
	})
}

func TestRpcAssert_IsTrue(t *testing.T) {
	t.Run("arguments is empty", func(t *testing.T) {
		assert := New(t)
		source := ""
		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o().IsTrue(); source = getFileLine(0) }()
		})).Equals(true, "\targuments is empty\n\t"+source+"\n")
	})

	t.Run("arguments is not true", func(t *testing.T) {
		assert := New(t)
		source := ""

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(nil).IsTrue(); source = getFileLine(0) }()
		})).Equals(true, "\t1st argument is not true\n\t"+source+"\n")

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(true, nil).IsTrue(); source = getFileLine(0) }()
		})).Equals(true, "\t2nd argument is not true\n\t"+source+"\n")
	})

	t.Run("test", func(t *testing.T) {
		assert := New(t)
		assert(true).IsTrue()
		assert(true, true).IsTrue()
	})
}

func TestRpcAssert_IsFalse(t *testing.T) {
	t.Run("arguments is empty", func(t *testing.T) {
		assert := New(t)
		source := ""
		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o().IsFalse(); source = getFileLine(0) }()
		})).Equals(true, "\targuments is empty\n\t"+source+"\n")
	})

	t.Run("arguments is not false", func(t *testing.T) {
		assert := New(t)
		source := ""

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(nil).IsFalse(); source = getFileLine(0) }()
		})).Equals(true, "\t1st argument is not false\n\t"+source+"\n")

		assert(testFailHelper(func(o func(_ ...interface{}) *Assert) {
			func() { o(false, nil).IsFalse(); source = getFileLine(0) }()
		})).Equals(true, "\t2nd argument is not false\n\t"+source+"\n")
	})

	t.Run("test", func(t *testing.T) {
		assert := New(t)
		assert(false).IsFalse()
		assert(false, false).IsFalse()
	})
}
