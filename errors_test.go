package errors

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"testing"

	"github.com/getlantern/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	replaceNumbers = regexp.MustCompile(`[0-9]+`)
	replaceArch    = regexp.MustCompile(`asm_[a-z0-9]+\.s`)
)

func TestNewWithCause(t *testing.T) {
	cause := buildCause()
	outer := New("Hello %v", cause)
	assert.Equal(t, "Hello World", outer.Error())
	assert.Equal(t, "Hello %v", outer.(Error).ErrorClean())
	require.IsType(t, (*wrappingError)(nil), outer, "Including an error arg should have resulted in a *wrappingError")
	assert.Equal(t,
		"github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)",
		replaceNumbers.ReplaceAllString(outer.(*wrappingError).data["error_location"].(string), "999"))
	assert.Equal(t, cause, outer.(*wrappingError).wrapped)

	// Make sure that stacktrace prints out okay
	buf := &bytes.Buffer{}
	print := outer.(Error).MultiLinePrinter()
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
  at runtime.goexit (asm_arch.s:999)
Caused by: World
  at github.com/getlantern/errors.buildCause (errors_test.go:999)
  at github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)
  at testing.tRunner (testing.go:999)
  at runtime.goexit (asm_arch.s:999)
Caused by: orld
Caused by: ld
  at github.com/getlantern/errors.buildSubSubCause (errors_test.go:999)
  at github.com/getlantern/errors.buildSubCause (errors_test.go:999)
  at github.com/getlantern/errors.buildCause (errors_test.go:999)
  at github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)
  at testing.tRunner (testing.go:999)
  at runtime.goexit (asm_arch.s:999)
Caused by: d
`

	assert.Equal(t,
		expected,
		replaceArch.ReplaceAllString(
			replaceNumbers.ReplaceAllString(buf.String(), "999"),
			"asm_arch.s",
		))
	assert.Equal(t, buildSubSubSubCause(), outer.(Error).RootCause())
}

func buildCause() error {
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

func TestFill(t *testing.T) {
	e := New("something happened").(*baseError)
	e2 := New("uh oh: %v", e).(*wrappingError)
	e3 := fmt.Errorf("hmm: %w", e2)
	e4 := New("umm: %v", e3).(*wrappingError)

	e4.data["name"] = "e4"
	e2.data["name"] = "e2"
	e.data["name"] = "e"
	e2.data["k"] = "v2"
	e.data["k"] = "v"
	e.data["a"] = "b"

	m := context.Map{}
	e4.Fill(m)
	require.Equal(t, "e4", m["name"])
	require.Equal(t, "v2", m["k"])
	require.Equal(t, "b", m["a"])
}

// Ensures that this package implements error unwrapping as described in:
// https://golang.org/pkg/errors/#pkg-overview
func TestUnwrapping(t *testing.T) {
	sampleUnwrapper := fmt.Errorf("%w", fmt.Errorf("something happened"))

	errNoCause := New("something happened")
	_, ok := errNoCause.(unwrapper)
	assert.False(t, ok, "error with no cause should not implement Unwrap method")
	wrappedNoCause := Wrap(errNoCause)
	_, ok = wrappedNoCause.(unwrapper)
	assert.False(t, ok, "wrapped error with no cause should not implement Unwrap method")

	errFromEOF := New("something happened: %v", io.EOF)
	assert.Implements(t, &sampleUnwrapper, errFromEOF)
	assert.True(t, errors.Is(errFromEOF, io.EOF))
	wrappedFromEOF := Wrap(errFromEOF)
	assert.Implements(t, &sampleUnwrapper, wrappedFromEOF)
	assert.True(t, errors.Is(wrappedFromEOF, io.EOF))

	addrErrHolder := new(net.AddrError)
	errFromAddrErr := New("something happend: %v", new(net.AddrError))
	assert.Implements(t, &sampleUnwrapper, errFromAddrErr)
	assert.True(t, errors.As(errFromAddrErr, &addrErrHolder))
	wrappedFromAddrErr := Wrap(errFromAddrErr)
	assert.Implements(t, &sampleUnwrapper, wrappedFromAddrErr)
	assert.True(t, errors.As(wrappedFromAddrErr, &addrErrHolder))
}
