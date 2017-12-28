package apachelog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func makeTestHandler(headers map[string]string, body string, code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(code)
		w.Write([]byte(body))
	}
}

func doFormat(t *testing.T, format string, r *http.Request, h http.Handler) string {
	var buf bytes.Buffer
	handler := NewHandler(format, h, &buf)
	server := httptest.NewServer(handler)
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		panic(err)
	}
	r.URL.Host = u.Host
	r.URL.Scheme = "http"
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return strings.TrimSpace(buf.String())
}

func newRequest(method string, url string, body string) *http.Request {
	r, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	return r
}

func TestSimple(t *testing.T) {
	for _, tc := range []struct {
		format string
		want   string
	}{
		{"%%", "%"},
		{"%h", "127.0.0.1"},
		{"%H", "HTTP/1.1"},
		{"%m", "GET"},
		{"%r", "GET / HTTP/1.1"},
		{"foo bar %% baz %m", "foo bar % baz GET"},
	} {
		got := doFormat(t, tc.format, newRequest("GET", "/", ""), makeTestHandler(nil, "", 200))
		if tc.want != got {
			t.Fatalf("For format string %q; want %q; got %q", tc.format, tc.want, got)
		}
	}
}

func TestRequestHeaders(t *testing.T) {
	for _, tc := range []struct {
		format  string
		headers map[string]string
		want    string
	}{
		{"%{Foobar}i", nil, ""},
		{"%{Foobar}i", map[string]string{"Foobar": "baz"}, "baz"},
		{"%u", nil, "-"},
		{"%u", map[string]string{"Remote-User": "alice"}, "alice"},
	} {
		r := newRequest("GET", "/", "")
		for k, v := range tc.headers {
			r.Header.Set(k, v)
		}
		got := doFormat(t, tc.format, r, makeTestHandler(nil, "", 200))
		if tc.want != got {
			t.Fatalf("With headers %v and format string %q; want %q; got %q", tc.headers, tc.format, tc.want, got)
		}
	}
}

func TestPathQuery(t *testing.T) {
	for _, tc := range []struct {
		format string
		path   string
		want   string
	}{
		{"%q", "/", ""},
		{"%q", "/path/to/resource?foo=bar&baz=quux", "?foo=bar&baz=quux"},
		{"%U", "/", "/"},
		{"%U", "/path/to/resource?foo=bar", "/path/to/resource"},
	} {
		got := doFormat(t, tc.format, newRequest("GET", tc.path, ""), makeTestHandler(nil, "", 200))
		if tc.want != got {
			t.Fatalf("For path %q and format string %q; want %q; got %q", tc.path, tc.format, tc.want, got)
		}
	}
}

func TestResponse(t *testing.T) {
	for _, tc := range []struct {
		format string
		body   string
		want   string
	}{
		{"%B", "", "0"},
		{"%B", "foo", "3"},
		{"%b", "", "-"},
		{"%b", "foo", "3"},
	} {
		got := doFormat(t, tc.format, newRequest("GET", "/", ""), makeTestHandler(nil, tc.body, 200))
		if tc.want != got {
			t.Fatalf("For response body %q and format string %q; want %q; got %q", tc.body, tc.format, tc.want, got)
		}
	}
}

func TestResponseHeaders(t *testing.T) {
	for _, tc := range []struct {
		format  string
		headers map[string]string
		want    string
	}{
		{"%{Foobar}o", nil, ""},
		{"%{Foobar}o", map[string]string{"Foobar": "baz"}, "baz"},
	} {
		got := doFormat(t, tc.format, newRequest("GET", "/", ""), makeTestHandler(tc.headers, "", 200))
		if tc.want != got {
			t.Fatalf("With response headers %v and format string %q; want %q; got %q", tc.headers, tc.format, tc.want, got)
		}
	}
}

func TestResponseCode(t *testing.T) {
	for _, tc := range []struct {
		format string
		code   int
		want   string
	}{
		{"%s", 200, "200"},
		{"%s", 400, "400"},
	} {
		got := doFormat(t, tc.format, newRequest("GET", "/", ""), makeTestHandler(nil, "", tc.code))
		if tc.want != got {
			t.Fatalf("With response code %d and format string %q; want %q; got %q", tc.code, tc.format, tc.want, got)
		}
	}
}

func TestTiming(t *testing.T) {
	const date = "2015-03-25T17:14:30-07:00"
	oldNow := now
	oldSince := since
	defer func() {
		now = oldNow
		since = oldSince
	}()
	ts, err := time.Parse(time.RFC3339, date)
	if err != nil {
		panic(err)
	}
	now = func() time.Time { return ts }

	for _, tc := range []struct {
		format string
		since  time.Duration
		want   string
	}{
		{"%t", 0, "[25/Mar/2015:17:14:30 -0700]"},
		{"%{02 Jan 06 15:04}t", 0, "[25 Mar 15 17:14]"},
		{"%D", 1234 * time.Millisecond, "1234000.0000"},
		{"%T", 1234 * time.Millisecond, "1.2340"},
	} {
		since = func(time.Time) time.Duration { return tc.since }
		got := doFormat(t, tc.format, newRequest("GET", "/", ""), makeTestHandler(nil, "", 200))
		if tc.want != got {
			t.Fatalf("With format string %q; want %q; got %q", tc.format, tc.want, got)
		}
	}
}
