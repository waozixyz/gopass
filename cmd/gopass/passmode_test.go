package main

import (
	"reflect"
	"testing"
)

func TestTranslatePassArguments(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "plain",
			in:   []string{"w", "example.org"},
			want: []string{"--copy", "--clear-after=1m30s", "--profile", "w", "example.org"},
		},
		{
			name: "all flags",
			in:   []string{"-vault", "/v", "-counter", "5", "-length", "20", "-symbols", "n", "s.org"},
			want: []string{"--copy", "--clear-after=1m30s", "--vault", "/v", "-C", "5", "-L", "20", "-s", "--profile", "n", "s.org"},
		},
		{
			name: "zero timeout keeps the clipboard",
			in:   []string{"-timeout", "0", "w", "example.org"},
			want: []string{"--copy", "--clear-after=0s", "--profile", "w", "example.org"},
		},
		{
			name: "read mode",
			in:   []string{"-read"},
			want: []string{"--copy", "--clear-after=1m30s", "--read-clipboard"},
		},
	}
	for _, test := range tests {
		got, ok := translatePassArguments(test.in)
		if !ok {
			t.Errorf("%s: translatePassArguments(%v) rejected the arguments", test.name, test.in)
			continue
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: got %v, want %v", test.name, got, test.want)
		}
	}
}

func TestTranslatePassArgumentsRejectsBadUsage(t *testing.T) {
	for _, arguments := range [][]string{
		{},
		{"w"},
		{"w", "example.org", "extra"},
		{"-length", "notanumber", "w", "example.org"},
	} {
		if got, ok := translatePassArguments(arguments); ok {
			t.Errorf("translatePassArguments(%v) = %v, want rejection", arguments, got)
		}
	}
}
