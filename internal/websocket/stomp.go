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

	nl := bytes.IndexByte(raw, '\n')
	if nl == -1 {
		return nil, fmt.Errorf("missing newline after command")
	}

	command := strings.TrimSpace(string(raw[:nl]))
	if command == "" {
		return nil, fmt.Errorf("empty command")
	}

	f := &Frame{Command: command, Headers: make(map[string]string)}
	rest := raw[nl+1:]
	if len(rest) == 0 {
		return f, nil
	}

	pos := 0
	for {
		nl := bytes.IndexByte(rest[pos:], '\n')
		if nl == -1 {
			if line := strings.TrimSpace(string(rest[pos:])); line != "" {
				f.addHeader(line)
			}
			break
		}
		line := strings.TrimSpace(string(rest[pos : pos+nl]))
		pos += nl + 1
		if line == "" {
			f.Body = rest[pos:]
			break
		}
		f.addHeader(line)
	}

	// honour content-length so bodies containing \n survive transport
	if cl := f.Headers["content-length"]; cl != "" {
		if n, err := strconv.Atoi(cl); err == nil && n >= 0 && n <= len(f.Body) {
			f.Body = f.Body[:n]
		}
	}

	return f, nil
}

func (f *Frame) addHeader(line string) {
	line = strings.TrimSuffix(line, "\r")
	idx := strings.Index(line, ":")
	if idx == -1 {
		return
	}
	k := strings.TrimSpace(line[:idx])
	v := strings.TrimSpace(line[idx+1:])
	f.Headers[k] = v
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
			"destination":  destination,
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

func ErrorFrameFor(frame *Frame, message string) *Frame {
	headers := map[string]string{"message": message}
	if dest := frame.Headers["destination"]; dest != "" {
		headers["destination"] = dest
	}
	if receipt := frame.Headers["receipt"]; receipt != "" {
		headers["receipt"] = receipt
	}
	return &Frame{Command: "ERROR", Headers: headers}
}

func ReceiptFrame(frame *Frame) *Frame {
	headers := map[string]string{}
	if dest := frame.Headers["destination"]; dest != "" {
		headers["destination"] = dest
	}
	if receipt := frame.Headers["receipt"]; receipt != "" {
		headers["receipt"] = receipt
	}
	return &Frame{Command: "RECEIPT", Headers: headers}
}
