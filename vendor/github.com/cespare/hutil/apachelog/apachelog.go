/*
Package apachelog is a library for logging the responses of an http.Handler. It uses formats and configuration
similar to the Apache HTTP server.

Format strings:
    %%          A literal %.
    %B          Size of the full HTTP response in bytes, excluding headers.
    %b          Size of the full HTTP response in bytes, excluding headers. This
                  is '-' rather than 0.
    %D          The time taken to serve the request, in microseconds. (Also see
                  %T.)
    %h          The client's IP address. (This is a best guess only -- see
                  hutil.RemoteIP.)
    %H          The request protocol.
    %{NAME}i    The contents of the request header called NAME.
    %m          The request method.
    %{NAME}o    The contents of the response header called NAME.
    %q          The query string (prepended with a ? if a query string exists;
                  otherwise an empty string).
    %r          First line of request (equivalent to '%m %U%q %H').
    %s          Response status code.
    %t          Time the request was received, formatted using ApacheTimeFormat
                  and surrounded by '[]'.
    %{FORMAT}t  Time the request was received, formatted using the supplied
                  time.Format string FORMAT and surrounded by '[]'.
    %T          The time taken to serve the request, in seconds. (Also see %D).
    %u          The remote user. May be bogus if the request was
                  unauthenticated.
    %U          The URL path requested, not including a query string.
*/
package apachelog

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cespare/hutil"
)

const (
	ApacheTimeFormat = `02/Jan/2006:15:04:05 -0700`

	// Predefined log formats.
	CommonLogFormat        = `%h - %u %t "%r" %s %b`
	CombinedLogFormat      = `%h - %u %t "%r" %s %b "%{Referer}i" "%{User-Agent}i"`
	RackCommonLoggerFormat = `%h - %u %{02/Jan/2006 15:04:05 -0700}t "%r" %s %b %T`
)

type parsedFormat struct {
	chunks []chunk
	buf    *bytes.Buffer

	neededReqHeaders  map[string]struct{}
	neededRespHeaders map[string]struct{}
}

func formatProvidedError(format byte) error {
	return fmt.Errorf("Format %%%c doesn't take a custom formatter.", format)
}

func newParsedFormat(format string) (*parsedFormat, error) {
	f := &parsedFormat{
		buf:               &bytes.Buffer{},
		neededReqHeaders:  make(map[string]struct{}),
		neededRespHeaders: make(map[string]struct{}),
	}
	chunks := []chunk{}

	// Add a newline to the format if it's not already provided.
	if format[len(format)-1] != '\n' {
		format = format + "\n"
	}

	var literal []byte
	var braceChunk []byte
	inBraceChunk := false // Whether we're in a brace-delimited formatter (e.g. %{NAME}i)
	escaped := false
outer:
	for _, c := range []byte(format) {
		if inBraceChunk {
			if c == '}' {
				inBraceChunk = false
			} else {
				braceChunk = append(braceChunk, c)
			}
			continue
		}
		if c == '%' {
			if escaped {
				literal = append(literal, '%')
			} else {
				if len(literal) > 0 {
					chunks = append(chunks, literalChunk(literal))
					literal = nil
				}
			}
			escaped = !escaped
			continue
		}
		if !escaped {
			literal = append(literal, c)
			continue
		}

		var ch chunk
		// First do the codes that can take a format chunk
		switch c {
		case '{':
			inBraceChunk = true
			continue outer
		case 'i':
			header := string(braceChunk)
			f.neededReqHeaders[header] = struct{}{}
			ch = reqHeaderChunk(header)
		case 'o':
			header := string(braceChunk)
			f.neededRespHeaders[header] = struct{}{}
			ch = respHeaderChunk(header)
		case 't':
			formatString := string(braceChunk)
			if braceChunk == nil {
				formatString = ApacheTimeFormat
			}
			ch = startTimeChunk(formatString)
		default:
			if braceChunk != nil {
				return nil, formatProvidedError(c)
			}
			switch c {
			case 'B':
				ch = responseBytesChunk(false)
			case 'b':
				ch = responseBytesChunk(true)
			case 'D':
				ch = responseTimeMicros
			case 'h':
				ch = clientIPChunk
			case 'H':
				ch = protoChunk
			case 'm':
				ch = methodChunk
			case 'q':
				ch = queryChunk
			case 'r':
				ch = requestLineChunk
			case 's':
				ch = statusChunk
			case 'T':
				ch = responseTimeSeconds
			case 'u':
				f.neededReqHeaders["Remote-User"] = struct{}{}
				ch = userChunk
			case 'U':
				ch = pathChunk
			default:
				return nil, fmt.Errorf("Unrecognized format code: %%%c", c)
			}
		}

		chunks = append(chunks, ch)
		escaped = false
		braceChunk = nil
	}

	if literal != nil {
		chunks = append(chunks, literalChunk(literal))
	}
	f.chunks = chunks
	return f, nil
}

func (f *parsedFormat) Write(r *record, out io.Writer) {
	f.buf.Reset()
	for _, c := range f.chunks {
		c(r, f.buf)
	}
	f.buf.WriteTo(out)
}

type handler struct {
	http.Handler
	pf *parsedFormat

	mu  sync.Mutex
	out io.Writer
}

func NewHandler(format string, h http.Handler, out io.Writer) http.Handler {
	pf, err := newParsedFormat(format)
	if err != nil {
		panic(err)
	}
	h2 := &handler{
		Handler: h,
		pf:      pf,
		out:     out,
	}
	return h2
}

func NewDefaultHandler(h http.Handler) http.Handler {
	return NewHandler(RackCommonLoggerFormat, h, os.Stderr)
}

var (
	now   = time.Now
	since = time.Since
)

func (h *handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	start := now()
	rec := &record{
		status: http.StatusOK, // Set to 200 to begin with because WriteHeader isn't called in the OK case.
	}
	rec.ip = hutil.RemoteIP(r).String()
	if len(h.pf.neededReqHeaders) > 0 {
		rec.reqHeaders = make(map[string]string)
		for header := range h.pf.neededReqHeaders {
			rec.reqHeaders[header] = r.Header.Get(header)
		}
	}
	rec.startTime = start
	rec.method = r.Method
	rec.path = r.URL.Path
	rec.query = r.URL.RawQuery
	rec.proto = r.Proto

	rec.ResponseWriter = rw
	h.Handler.ServeHTTP(rec, r)

	rec.elapsed = since(start)
	if len(h.pf.neededRespHeaders) > 0 {
		rec.respHeaders = make(map[string]string)
		for header := range h.pf.neededRespHeaders {
			rec.respHeaders[header] = rw.Header().Get(header)
		}
	}

	h.mu.Lock()
	h.pf.Write(rec, h.out)
	h.mu.Unlock()
}

// Only the necessary fields will be filled out.
type record struct {
	http.ResponseWriter

	ip            string
	responseBytes int64
	startTime     time.Time
	elapsed       time.Duration
	proto         string
	reqHeaders    map[string]string // Just the ones needed for the format, or nil if there are none
	method        string
	respHeaders   map[string]string
	query         string
	status        int
	path          string
}

// Write proxies to the underlying ResponseWriter's Write method, while recording response size.
func (r *record) Write(p []byte) (int, error) {
	written, err := r.ResponseWriter.Write(p)
	r.responseBytes += int64(written)
	return written, err
}

func (r *record) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
