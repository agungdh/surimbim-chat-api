package websocket

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type Frame struct {
	Command string
	Headers map[string]string
	Body    []byte
}

func ParseFrame(raw []byte) (*Frame, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty frame")
	}

	raw = bytes.TrimSuffix(raw, []byte{0})

	parts := strings.SplitN(string(raw), "\n", 2)
	if len(parts) == 0 {
		return nil, fmt.Errorf("no command")
	}

	f := &Frame{
		Command: strings.TrimSpace(parts[0]),
		Headers: make(map[string]string),
	}

	if len(parts) == 1 {
		return f, nil
	}

	remaining := parts[1]
	bodyIdx := strings.Index(remaining, "\n\n")
	if bodyIdx != -1 {
		headerBlock := remaining[:bodyIdx]
		f.parseHeaders(headerBlock)
		f.Body = []byte(remaining[bodyIdx+2:])
	} else {
		f.parseHeaders(remaining)
	}

	return f, nil
}

func (f *Frame) parseHeaders(block string) {
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		f.Headers[k] = v
	}
}

func (f *Frame) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteString(f.Command)
	buf.WriteByte('\n')
	for k, v := range f.Headers {
		buf.WriteString(k)
		buf.WriteByte(':')
		buf.WriteString(v)
		buf.WriteByte('\n')
	}
	buf.WriteByte('\n')
	buf.Write(f.Body)
	buf.WriteByte(0)
	return buf.Bytes()
}

func ConnectFrame(userID int64) *Frame {
	return &Frame{
		Command: "CONNECTED",
		Headers: map[string]string{
			"version":    "1.2",
			"heart-beat": "0,0",
			"user-id":    strconv.FormatInt(userID, 10),
		},
	}
}

func MessageFrame(destination string, body []byte) *Frame {
	return &Frame{
		Command: "MESSAGE",
		Headers: map[string]string{
			"destination": destination,
			"content-type": "application/json",
		},
		Body: body,
	}
}

func ErrorFrame(message string) *Frame {
	return &Frame{
		Command: "ERROR",
		Headers: map[string]string{
			"message": message,
		},
	}
}
