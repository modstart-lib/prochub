//go:build windows

package service

import (
	"testing"
	"unicode/utf16"
)

func TestParseWSLDistros(t *testing.T) {
	out := "  NAME            STATE           VERSION\n" +
		"* Ubuntu          Running         2\n" +
		"  Debian          Stopped         2\n" +
		"  kali-linux      Stopped         1\n"

	distros := parseWSLDistros(out)
	if len(distros) != 3 {
		t.Fatalf("expected 3 distros, got %d", len(distros))
	}

	if distros[0].Name != "Ubuntu" || distros[0].State != "Running" || distros[0].Version != "2" || !distros[0].Default {
		t.Errorf("unexpected default distro: %+v", distros[0])
	}
	if distros[1].Name != "Debian" || distros[1].Default {
		t.Errorf("unexpected second distro: %+v", distros[1])
	}
	if distros[2].Name != "kali-linux" || distros[2].Version != "1" {
		t.Errorf("unexpected third distro: %+v", distros[2])
	}
}

func TestParseWSLDistrosSkipsHeaderAndBlankLines(t *testing.T) {
	if distros := parseWSLDistros("\n\n  NAME   STATE   VERSION\n\n"); len(distros) != 0 {
		t.Errorf("expected no distros, got %+v", distros)
	}
}

func TestDecodeUTF16LE(t *testing.T) {
	input := "WSL version: 2.7.14.0\r\n"
	codes := utf16.Encode([]rune(input))
	raw := make([]byte, 0, len(codes)*2)
	for _, c := range codes {
		raw = append(raw, byte(c), byte(c>>8))
	}

	if got := decodeUTF16LE(raw); got != input {
		t.Errorf("decodeUTF16LE() = %q, want %q", got, input)
	}
	if got := decodeUTF16LE([]byte("plain ascii")); got != "plain ascii" {
		t.Errorf("decodeUTF16LE(ascii) = %q", got)
	}
	if got := decodeUTF16LE(nil); got != "" {
		t.Errorf("decodeUTF16LE(nil) = %q, want empty", got)
	}
}
