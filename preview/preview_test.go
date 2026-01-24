package preview

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testMaxChars   = 35
	testMaxInspect = 4096
)

func TestPreview_Empty(t *testing.T) {
	t.Log("empty input")
	out := Preview(nil, testMaxChars, testMaxInspect)
	t.Logf("preview: %q", out)
	require.Equal(t, "", out)

	t.Log("empty slice")
	out = Preview([]byte{}, testMaxChars, testMaxInspect)
	t.Logf("preview: %q", out)
	require.Equal(t, "", out)

	t.Log("whitespace-only input")
	out = Preview([]byte("   \n\t  "), testMaxChars, testMaxInspect)
	t.Logf("preview: %q", out)
	require.Equal(t, "", out)
}

func TestPreview_SingleLine(t *testing.T) {
	in := []byte("hello world")
	out := Preview(in, testMaxChars, testMaxInspect)

	t.Logf("input: %q", in)
	t.Logf("preview: %q", out)

	require.Equal(t, "hello world", out)
}

func TestPreview_MultilineTextCollapsesWhitespace(t *testing.T) {
	in := []byte("hello\nworld\tthis   is\r\nnice")
	out := Preview(in, testMaxChars, testMaxInspect)

	t.Logf("input: %q", in)
	t.Logf("preview: %q", out)

	require.Equal(t, "hello world this is nice", out)
}

func TestPreview_TruncatesRunesNotBytes(t *testing.T) {
	in := []byte("hello 🙂🙂🙂🙂🙂 world")
	out := Preview(in, 10, testMaxInspect)

	t.Logf("input: %q", in)
	t.Logf("preview (maxChars=10): %q", out)

	require.Equal(t, "hello 🙂🙂🙂…", out)
}

func TestPreview_SingleLineJSON_RemainsCompact(t *testing.T) {
	in := []byte(`{"a": 1, "b": "two"}`)
	out := Preview(in, testMaxChars, testMaxInspect)

	t.Logf("input JSON: %s", in)
	t.Logf("preview: %s", out)

	require.Equal(t, `{"a":1,"b":"two"}`, out)
}

func TestPreview_MultilineJSON_CompactsSoNotJustBrace(t *testing.T) {
	in := []byte("{\n  \"a\": 1,\n  \"b\": {\n    \"c\": 2\n  }\n}\n")
	out := Preview(in, testMaxChars, testMaxInspect)

	t.Log("multiline JSON input")
	t.Logf("preview: %s", out)

	require.NotEqual(t, "{", out)
	require.NotEqual(t, "{…", out)
	require.True(t,
		strings.HasPrefix(out, `{"a":1,`) || strings.HasPrefix(out, `{"a":1`),
		"expected compact json prefix, got %q", out,
	)
}

func TestPreview_JSONArray_Compacts(t *testing.T) {
	in := []byte("[\n  1,\n  2,\n  3\n]\n")
	out := Preview(in, testMaxChars, testMaxInspect)

	t.Log("JSON array input")
	t.Logf("preview: %s", out)

	require.Equal(t, "[1,2,3]", out)
}

func TestPreview_TruncationAddsEllipsis(t *testing.T) {
	in := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	out := Preview(in, testMaxChars, testMaxInspect)

	t.Logf("input length=%d", len(in))
	t.Logf("preview: %q", out)

	require.Equal(t, "abcdefghijklmnopqrstuvwxyz01234567…", out)
}

func TestPreview_MaxCharsOne(t *testing.T) {
	in := []byte("hello")
	out := Preview(in, 1, testMaxInspect)

	t.Logf("preview (maxChars=1): %q", out)

	require.Equal(t, "…", out)
}

func TestPreview_InvalidUTF8_BinaryPreview(t *testing.T) {
	in := []byte{0xff, 0xfe, 0xfd, 0x00, 0x01, 0x02, 0x03}
	out := Preview(in, testMaxChars, testMaxInspect)

	t.Logf("binary input: %v", in)
	t.Logf("preview: %q", out)

	require.True(t, strings.HasPrefix(out, "<binary "), "expected binary preview, got %q", out)
	require.Contains(t, out, "B>")
}

func TestPreview_ValidUTF8ButNonPrintable_BinaryPreview(t *testing.T) {
	in := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	out := Preview(in, testMaxChars, testMaxInspect)

	t.Logf("control-heavy input: %v", in)
	t.Logf("preview: %q", out)

	require.True(t, strings.HasPrefix(out, "<binary "), "expected binary preview, got %q", out)
}

func TestPreview_RespectsMaxInspect(t *testing.T) {
	prefix := "{\n  \"a\": 1,\n  \"b\": \""
	suffix := strings.Repeat("x", 5000) + "\"\n}\n"
	in := []byte(prefix + suffix)

	out := Preview(in, testMaxChars, len(prefix)+10)

	t.Logf("maxInspect=%d", len(prefix)+10)
	t.Logf("preview: %q", out)

	require.NotEmpty(t, out)
	require.NotEqual(t, "{", out)
	require.Contains(t, out, `"a":`)
}