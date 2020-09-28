package errors

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/getlantern/context"
	"github.com/getlantern/hidden"
	"github.com/getlantern/ops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	replaceNumbers = regexp.MustCompile("[0-9]+")
)

func TestFull(t *testing.T) {
	var firstErr Error

	// Iterate past the size of the hidden buffer
	for i := 0; i < len(hiddenErrors)*2; i++ {
		op := ops.Begin("op1").Set("ca", 100).Set("cd", 100)
		e := New("Hello %v", "There").Op("My Op").With("DaTa_1", 1)
		op.End()
		if firstErr == nil {
			firstErr = e
		}
		assert.Equal(t, "Hello There", e.Error()[:11])
		op = ops.Begin("op2").Set("ca", 200).Set("cb", 200).Set("cc", 200)
		e3 := Wrap(fmt.Errorf("I'm wrapping your text: %w", e)).Op("outer op").With("dATA+1", i).With("cb", 300)
		op.End()
		require.IsType(t, (*wrappingError)(nil), e3, "wrapping an error with cause should have resulted in a *wrappingError")
		assert.Equal(t, e, e3.(*wrappingError).wrapped, "Wrapping a regular error should have extracted the contained *Error")
		m := make(context.Map)
		e3.Fill(m)
		assert.Equal(t, i, m["data_1"], "Error's data should dominate all")
		assert.Equal(t, 200, m["ca"], "Error's context should dominate cause")
		assert.Equal(t, 300, m["cb"], "Error's data should dominate its context")
		assert.Equal(t, 200, m["cc"], "Error's context should come through")
		assert.Equal(t, 100, m["cd"], "Cause's context should come through")
		assert.Equal(t, "My Op", e.(*baseError).data["error_op"], "Op should be available from cause")

		for _, call := range e3.(*wrappingError).callStack {
			t.Logf("at %v", call)
		}
	}

	e3 := Wrap(fmt.Errorf("I'm wrapping your text: %v", firstErr)).With("a", 2)
	// TODO: is this correct? (replaces commented line below)
	require.IsType(t, (*baseError)(nil), e3, "Wrapping an *Error that's no longer buffered should have resulted in a *baseError")
	// assert.Nil(t, e3.(*baseError).cause, "Wrapping an *Error that's no longer buffered should have yielded no cause")
}

func TestNewWithCause(t *testing.T) {
	cause := buildCause()
	outer := New("Hello %v", cause)
	assert.Equal(t, "Hello World", hidden.Clean(outer.Error()))
	assert.Equal(t, "Hello %v", outer.ErrorClean())
	require.IsType(t, (*wrappingError)(nil), outer, "Including an error arg should have resulted in a *wrappingError")
	assert.Equal(t,
		"github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)",
		replaceNumbers.ReplaceAllString(outer.(*wrappingError).data["error_location"].(string), "999"))
	assert.Equal(t, cause, outer.(*wrappingError).wrapped)

	// Make sure that stacktrace prints out okay
	buf := &bytes.Buffer{}
	print := outer.MultiLinePrinter()
	for {
		more := print(buf)
		buf.WriteByte('\n')
		if !more {
			break
		}
	}
	expected := `Hello World
  at github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)
  at testing.tRunner (testing.go:999)
  at runtime.goexit (asm_amd999.s:999)
Caused by: World
  at github.com/getlantern/errors.buildCause (errors_test.go:999)
  at github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)
  at testing.tRunner (testing.go:999)
  at runtime.goexit (asm_amd999.s:999)
Caused by: orld
Caused by: ld
  at github.com/getlantern/errors.buildSubSubCause (errors_test.go:999)
  at github.com/getlantern/errors.buildSubCause (errors_test.go:999)
  at github.com/getlantern/errors.buildCause (errors_test.go:999)
  at github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)
  at testing.tRunner (testing.go:999)
  at runtime.goexit (asm_amd999.s:999)
Caused by: d
`

	assert.Equal(t, expected, replaceNumbers.ReplaceAllString(hidden.Clean(buf.String()), "999"))
	assert.Equal(t, buildSubSubSubCause(), outer.RootCause())
}

func buildCause() Error {
	return New("W%v", buildSubCause())
}

func buildSubCause() error {
	return fmt.Errorf("or%w", buildSubSubCause())
}

func buildSubSubCause() error {
	return New("l%v", buildSubSubSubCause())
}

func buildSubSubSubCause() error {
	return fmt.Errorf("d")
}

func TestWrapNil(t *testing.T) {
	assert.Nil(t, doWrapNil())
}

func doWrapNil() error {
	return Wrap(nil)
}

func TestHiddenWithCause(t *testing.T) {
	e1 := fmt.Errorf("I failed %v", "dude")
	e2 := New("I wrap: %v", e1)
	e3 := fmt.Errorf("Hiding %v", e2)
	// clear hidden buffer
	hiddenErrors = make([]*baseError, 100)
	e4 := Wrap(e3)
	e5 := New("I'm really outer: %v", e4)

	buf := &bytes.Buffer{}
	print := e5.MultiLinePrinter()
	for {
		more := print(buf)
		buf.WriteByte('\n')
		if !more {
			break
		}
	}
	// We're not asserting the output because we're just making sure that printing
	// doesn't panic. If we get to this point without panicking, we're happy.
}

// TODO: delete or remove redundant tests elsewhere
// TODO: remove print statements
func TestMultilinePrinter(t *testing.T) {
	e := New("an error occurred")
	require.IsType(t, (*baseError)(nil), e, "error without cause should be a *baseError")
	p := e.MultiLinePrinter()
	buf := new(bytes.Buffer)
	for p(buf) {
		fmt.Fprintln(buf)
	}

	e2 := New("something happened: %v", e)
	require.IsType(t, (*wrappingError)(nil), e2, "error wrapping another should be a *wrappingError")
	p2 := e2.MultiLinePrinter()
	buf = new(bytes.Buffer)
	for p2(buf) {
		fmt.Fprintln(buf)
	}
}
