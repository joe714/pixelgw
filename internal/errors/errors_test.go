package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	err := New(500, "test error")
	require.NotNil(t, err)

	assert.Equal(t, "test error", err.Error())

	// Verify it implements CodedError
	coded, ok := err.(CodedError)
	require.True(t, ok, "error should implement CodedError")
	assert.Equal(t, int32(500), coded.Code())
}

func TestCerrorError(t *testing.T) {
	err := New(123, "custom message")
	assert.Equal(t, "custom message", err.Error())
}

func TestCerrorCode(t *testing.T) {
	tests := []struct {
		name string
		code int32
	}{
		{"zero code", 0},
		{"positive code", 200},
		{"large code", 9999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, "test")
			coded := err.(CodedError)
			assert.Equal(t, tt.code, coded.Code())
		})
	}
}

func TestWrap(t *testing.T) {
	base := New(100, "base error")
	wrapped := Wrap(base, "wrapped: %s", "additional info")

	assert.Equal(t, "wrapped: additional info", wrapped.Error())
}

func TestWerrorUnwrap(t *testing.T) {
	base := New(100, "base error")
	wrapped := Wrap(base, "wrapped error")

	// Unwrap should return the base error
	unwrapped := wrapped.(interface{ Unwrap() error }).Unwrap()
	assert.Equal(t, base, unwrapped)
}

func TestCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int32
	}{
		{
			name:     "coded error",
			err:      New(500, "coded"),
			wantCode: 500,
		},
		{
			name:     "wrapped coded error",
			err:      Wrap(New(404, "not found"), "wrapped"),
			wantCode: 404,
		},
		{
			name:     "standard error returns default",
			err:      fmt.Errorf("standard error"),
			wantCode: 1000,
		},
		{
			name:     "nil error returns default",
			err:      nil,
			wantCode: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := Code(tt.err)
			assert.Equal(t, tt.wantCode, code)
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int32
		wantMsg  string
	}{
		{
			name:     "ChannelExists",
			err:      ChannelExists,
			wantCode: 1001,
			wantMsg:  "channel exists",
		},
		{
			name:     "ChannelNotFound",
			err:      ChannelNotFound,
			wantCode: 1002,
			wantMsg:  "channel not found",
		},
		{
			name:     "AppIndexOutOfRange",
			err:      AppIndexOutOfRange,
			wantCode: 1011,
			wantMsg:  "index out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
			assert.Equal(t, tt.wantCode, Code(tt.err))
		})
	}
}

func TestWrapFormatsCorrectly(t *testing.T) {
	base := ChannelNotFound
	wrapped := Wrap(base, "channel %s (uuid: %s)", "test-channel", "abc-123")

	assert.Equal(t, "channel test-channel (uuid: abc-123)", wrapped.Error())

	// Should still be able to extract the code from the wrapped error
	assert.Equal(t, int32(1002), Code(wrapped))
}

func TestDoubleWrap(t *testing.T) {
	base := New(100, "base")
	wrap1 := Wrap(base, "first wrap")
	wrap2 := Wrap(wrap1, "second wrap")

	assert.Equal(t, "second wrap", wrap2.Error())
	assert.Equal(t, int32(100), Code(wrap2))
}
