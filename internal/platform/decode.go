package platform

import (
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// DecodeOutput converts raw bytes captured from a child console program into a
// UTF-8 string.
//
// Windows console programs (notably .NET ones) write text using the console
// output code page, or the system ANSI code page when no console is attached,
// rather than UTF-8. Output redirected into a pipe therefore arrives as legacy
// encoded bytes (e.g. CP936/GBK on Simplified Chinese systems). Storing those
// bytes as-is renders as mojibake once the text is JSON-encoded, because every
// invalid UTF-8 sequence is replaced with U+FFFD ("版本" -> "�汾"). Valid
// UTF-8 passes through untouched, so UTF-8 programs (Go, Node, ...) are
// unaffected.
func DecodeOutput(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if utf8.Valid(b) {
		return string(b)
	}
	return decodeLegacy(b, activeCodePage())
}

// decodeLegacy transcodes bytes encoded with a Windows legacy code page into
// UTF-8, keeping the raw bytes when the code page is unknown.
func decodeLegacy(b []byte, codePage uint32) string {
	enc := legacyEncoding(codePage)
	if enc == nil {
		return string(b)
	}
	out, _, err := transform.Bytes(enc.NewDecoder(), b)
	if err != nil {
		return string(b)
	}
	return string(out)
}

// legacyEncoding maps a Windows code page number to its decoder. Only the
// pages commonly seen on desktop systems are covered; anything else falls back
// to the raw bytes.
func legacyEncoding(codePage uint32) encoding.Encoding {
	switch codePage {
	case 936:
		return simplifiedchinese.GBK
	case 950:
		return traditionalchinese.Big5
	case 932:
		return japanese.ShiftJIS
	case 949:
		return korean.EUCKR
	case 1250:
		return charmap.Windows1250
	case 1251:
		return charmap.Windows1251
	case 1252:
		return charmap.Windows1252
	case 1253:
		return charmap.Windows1253
	case 1254:
		return charmap.Windows1254
	case 1255:
		return charmap.Windows1255
	case 1256:
		return charmap.Windows1256
	case 1257:
		return charmap.Windows1257
	case 1258:
		return charmap.Windows1258
	default:
		return nil
	}
}
