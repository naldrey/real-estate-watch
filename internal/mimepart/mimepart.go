// Package mimepart extracts a decoded text part from a raw email message,
// handling both single-part and multipart messages, and the transfer
// encoding / charset conversion in between.
package mimepart

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset" // registers non-UTF-8 charset decoding (e.g. iso-8859-1)
)

// ErrPartNotFound is returned (wrapped) by ExtractHTML/ExtractPlainText when
// the message has no part of the requested media type. Callers can check for
// it with errors.Is to distinguish "this message isn't the shape we parse"
// from a genuine read/decode failure.
var ErrPartNotFound = errors.New("mime part not found")

// ExtractHTML returns the decoded text/html content of raw.
func ExtractHTML(raw []byte) (string, error) {
	return extract(raw, "text/html")
}

// ExtractPlainText returns the decoded text/plain content of raw.
func ExtractPlainText(raw []byte) (string, error) {
	return extract(raw, "text/plain")
}

// Subject returns the message's decoded Subject header (RFC 2047
// encoded-words and charsets are resolved to UTF-8).
func Subject(raw []byte) (string, error) {
	m, err := message.Read(bytes.NewReader(raw))
	if err != nil && !message.IsUnknownCharset(err) {
		return "", fmt.Errorf("read message: %w", err)
	}

	subject, err := m.Header.Text("Subject")
	if err != nil && !message.IsUnknownCharset(err) {
		return "", fmt.Errorf("read subject: %w", err)
	}

	return subject, nil
}

func extract(raw []byte, mediaType string) (string, error) {
	m, err := message.Read(bytes.NewReader(raw))
	if err != nil && !message.IsUnknownCharset(err) {
		return "", fmt.Errorf("read message: %w", err)
	}

	content, err := findPart(m, mediaType)
	if err != nil {
		return "", err
	}
	if content == "" {
		return "", fmt.Errorf("no %s part found: %w", mediaType, ErrPartNotFound)
	}

	return content, nil
}

func findPart(e *message.Entity, mediaType string) (string, error) {
	if mr := e.MultipartReader(); mr != nil {
		for {
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return "", fmt.Errorf("read part: %w", err)
			}

			content, err := findPart(part, mediaType)
			if err != nil {
				return "", err
			}
			if content != "" {
				return content, nil
			}
		}
		return "", nil
	}

	partType, _, err := e.Header.ContentType()
	if err != nil || !strings.HasPrefix(partType, mediaType) {
		return "", nil
	}

	b, err := io.ReadAll(e.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	return string(b), nil
}
