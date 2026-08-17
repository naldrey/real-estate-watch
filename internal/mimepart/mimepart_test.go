package mimepart

import (
	"errors"
	"testing"
)

func TestExtractHTML_SinglePartQuotedPrintable(t *testing.T) {
	raw := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"<p>Caf=C3=A9</p>")

	got, err := ExtractHTML(raw)
	if err != nil {
		t.Fatalf("ExtractHTML() returned error: %v", err)
	}
	if want := "<p>Café</p>"; got != want {
		t.Errorf("ExtractHTML() = %q, want %q", got, want)
	}
}

func TestExtractHTML_MultipartAlternativePicksHTML(t *testing.T) {
	raw := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=BOUNDARY\r\n" +
		"\r\n" +
		"--BOUNDARY\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"Plain text version\r\n" +
		"--BOUNDARY\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		"<p>HTML version</p>\r\n" +
		"--BOUNDARY--\r\n")

	got, err := ExtractHTML(raw)
	if err != nil {
		t.Fatalf("ExtractHTML() returned error: %v", err)
	}
	if want := "<p>HTML version</p>"; got != want {
		t.Errorf("ExtractHTML() = %q, want %q", got, want)
	}
}

func TestExtractHTML_ConvertsISO8859_1ToUTF8(t *testing.T) {
	// 0xF3 is the ISO-8859-1 byte for 'ó'.
	raw := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=iso-8859-1\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"<p>Arag=F3n</p>")

	got, err := ExtractHTML(raw)
	if err != nil {
		t.Fatalf("ExtractHTML() returned error: %v", err)
	}
	if want := "<p>Aragón</p>"; got != want {
		t.Errorf("ExtractHTML() = %q, want %q", got, want)
	}
}

func TestExtractHTML_NoHTMLPartReturnsError(t *testing.T) {
	raw := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"just text")

	_, err := ExtractHTML(raw)
	if err == nil {
		t.Fatal("ExtractHTML() returned nil error for a message with no html part")
	}
	if !errors.Is(err, ErrPartNotFound) {
		t.Errorf("ExtractHTML() error = %v, want it to wrap ErrPartNotFound", err)
	}
}

func TestExtractPlainText_MultipartAlternativePicksPlainText(t *testing.T) {
	raw := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=BOUNDARY\r\n" +
		"\r\n" +
		"--BOUNDARY\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"Plain text version\r\n" +
		"--BOUNDARY\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		"<p>HTML version</p>\r\n" +
		"--BOUNDARY--\r\n")

	got, err := ExtractPlainText(raw)
	if err != nil {
		t.Fatalf("ExtractPlainText() returned error: %v", err)
	}
	if want := "Plain text version"; got != want {
		t.Errorf("ExtractPlainText() = %q, want %q", got, want)
	}
}

func TestExtractPlainText_NoPlainTextPartReturnsError(t *testing.T) {
	raw := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		"<p>only html</p>")

	if _, err := ExtractPlainText(raw); err == nil {
		t.Fatal("ExtractPlainText() returned nil error for a message with no plain text part")
	}
}

func TestSubject_PlainASCII(t *testing.T) {
	raw := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"Subject: 6 novedades en Barcelona\r\n" +
		"\r\n" +
		"body")

	got, err := Subject(raw)
	if err != nil {
		t.Fatalf("Subject() returned error: %v", err)
	}
	if want := "6 novedades en Barcelona"; got != want {
		t.Errorf("Subject() = %q, want %q", got, want)
	}
}

func TestSubject_DecodesRFC2047EncodedWords(t *testing.T) {
	raw := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"Subject: =?UTF-8?Q?=C2=A1Nuevo_piso!?=\r\n" +
		"\r\n" +
		"body")

	got, err := Subject(raw)
	if err != nil {
		t.Fatalf("Subject() returned error: %v", err)
	}
	if want := "¡Nuevo piso!"; got != want {
		t.Errorf("Subject() = %q, want %q", got, want)
	}
}
