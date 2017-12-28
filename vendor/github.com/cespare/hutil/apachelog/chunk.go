package apachelog

import (
	"bytes"
	"fmt"
	"net/http"
)

// A chunk is a function that knows how to write some chunk of a record into a bytes.Buffer.
type chunk func(*record, *bytes.Buffer)

func literalChunk(literal []byte) chunk {
	return func(r *record, buf *bytes.Buffer) {
		buf.Write(literal)
	}
}

func clientIPChunk(r *record, buf *bytes.Buffer) { buf.WriteString(r.ip) }

func userChunk(r *record, buf *bytes.Buffer) {
	h := r.reqHeaders["Remote-User"]
	if h == "" {
		h = "-"
	}
	buf.WriteString(h)
}

func reqHeaderChunk(header string) chunk {
	h := http.CanonicalHeaderKey(header)
	return func(r *record, buf *bytes.Buffer) { buf.WriteString(r.reqHeaders[h]) }
}

func respHeaderChunk(header string) chunk {
	h := http.CanonicalHeaderKey(header)
	return func(r *record, buf *bytes.Buffer) { buf.WriteString(r.respHeaders[h]) }
}

func startTimeChunk(format string) chunk {
	return func(r *record, buf *bytes.Buffer) {
		fmt.Fprintf(buf, "[%s]", r.startTime.Format(format))
	}
}

func prefixRawQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	return "?" + rawQuery
}

func methodChunk(r *record, buf *bytes.Buffer) { buf.WriteString(r.method) }
func pathChunk(r *record, buf *bytes.Buffer)   { buf.WriteString(r.path) }
func queryChunk(r *record, buf *bytes.Buffer)  { buf.WriteString(prefixRawQuery(r.query)) }
func protoChunk(r *record, buf *bytes.Buffer)  { buf.WriteString(r.proto) }

func requestLineChunk(r *record, buf *bytes.Buffer) {
	fmt.Fprintf(buf, "%s %s%s %s", r.method, r.path, prefixRawQuery(r.query), r.proto)
}

func statusChunk(r *record, buf *bytes.Buffer) { fmt.Fprintf(buf, "%d", r.status) }

func responseBytesChunk(replaceWithDash bool) chunk {
	return func(r *record, buf *bytes.Buffer) {
		size := r.responseBytes
		if size == 0 && replaceWithDash {
			buf.WriteString("-")
		} else {
			fmt.Fprintf(buf, "%d", size)
		}
	}
}

func responseTimeSeconds(r *record, buf *bytes.Buffer) { fmt.Fprintf(buf, "%.4f", r.elapsed.Seconds()) }

func responseTimeMicros(r *record, buf *bytes.Buffer) {
	fmt.Fprintf(buf, "%.4f", float64(r.elapsed.Nanoseconds())/1000.0)
}
