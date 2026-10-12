package platform

import "testing"

func TestDecodeLegacyGBK(t *testing.T) {
	// GBK bytes for "版本"; read as UTF-8 they render as the mojibake "�汾".
	gbk := []byte{0xB0, 0xE6, 0xB1, 0xBE}
	if got := decodeLegacy(gbk, 936); got != "版本" {
		t.Fatalf("decodeLegacy(gbk, 936) = %q, want %q", got, "版本")
	}
}

func TestDecodeOutputKeepsValidUTF8(t *testing.T) {
	in := "正常中文 log 日志"
	if got := DecodeOutput([]byte(in)); got != in {
		t.Fatalf("DecodeOutput(utf8) = %q, want %q", got, in)
	}
}

func TestDecodeOutputKeepsASCII(t *testing.T) {
	if got := DecodeOutput([]byte("hello world")); got != "hello world" {
		t.Fatalf("DecodeOutput(ascii) = %q", got)
	}
}

func TestDecodeOutputEmpty(t *testing.T) {
	if got := DecodeOutput(nil); got != "" {
		t.Fatalf("DecodeOutput(nil) = %q, want empty", got)
	}
}

func TestDecodeLegacyUnknownCodePageKeepsBytes(t *testing.T) {
	gbk := []byte{0xB0, 0xE6, 0xB1, 0xBE}
	if got := decodeLegacy(gbk, 437); got != string(gbk) {
		t.Fatalf("decodeLegacy(unknown) = %q, want raw bytes", got)
	}
}
