package pass

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestDeriveKeyKnownResults(t *testing.T) {
	tests := []struct {
		password string
		salt     string
		want     string
	}{
		{"password", "salt", "0394a2ede332c9a13eb82e9b24631604c31df978b4e2f0fbd2c549944f9d79a5"},
		{"", "examplealice1", "caa46554f5a676b76c15b368c655b0b24eaaad8595ef919999785c68e60fd5f5"},
	}
	for _, test := range tests {
		got := hex.EncodeToString(deriveKey([]byte(test.password), []byte(test.salt)))
		if got != test.want {
			t.Errorf("deriveKey(%q, %q) = %s, want %s", test.password, test.salt, got, test.want)
		}
	}
}

func TestGenerateKnownResults(t *testing.T) {
	tests := []struct {
		name             string
		site, login, key string
		options          Options
		want             string
	}{
		{
			name: "defaults",
			site: "example.com", login: "alice", key: "correct horse battery staple",
			options: DefaultOptions(), want: "&vLf44D'/cSkP-_8",
		},
		{
			name: "counter and length",
			site: "service.test", login: "person@example.net", key: "master",
			options: Options{Length: 20, Counter: 2, Lowercase: true, Uppercase: true, Digits: true, Symbols: true},
			want:    "j:x_Lo5X_jL0w%ez`1be",
		},
		{
			name: "utf8 inputs and two classes",
			site: "δοκιμή.example", login: "ユーザー", key: "pässword",
			options: Options{Length: 12, Counter: 7, Lowercase: true, Digits: true},
			want:    "ioh5o2mhyghv",
		},
		{
			name: "excluded characters",
			site: "example.com", login: "alice", key: "correct horse battery staple",
			options: Options{Length: 24, Counter: 1, Lowercase: true, Uppercase: true, Digits: true, Symbols: true, Exclude: "0Ool1I!|"},
			want:    "7,a.Cp}YnF'ee7HqbX#PQhgH",
		},
		{
			name: "empty master and zero counter",
			site: "x", login: "y", key: "",
			options: Options{Length: 4, Counter: 0, Uppercase: true, Digits: true},
			want:    "9O7C",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Generate(test.site, test.login, test.key, test.options)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("Generate() = %q, want %q", got, test.want)
			}
			independent := referenceGenerate(test.site, test.login, test.key, test.options)
			if got != independent {
				t.Fatalf("Generate() = %q, separate reference = %q", got, independent)
			}
		})
	}
}

func TestGeneratedPasswordMeetsRequestedShape(t *testing.T) {
	options := Options{
		Length:    40,
		Counter:   3,
		Lowercase: true,
		Uppercase: true,
		Digits:    true,
		Symbols:   true,
		Exclude:   "abcXYZ019!@",
	}
	got, err := Generate("host", "account", "secret", options)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != options.Length {
		t.Fatalf("length = %d, want %d", len(got), options.Length)
	}
	for _, excluded := range options.Exclude {
		if strings.ContainsRune(got, excluded) {
			t.Errorf("result contains excluded character %q", excluded)
		}
	}
	for name, set := range map[string]string{
		"lowercase": removeCharacters(LowercaseCharacters, options.Exclude),
		"uppercase": removeCharacters(UppercaseCharacters, options.Exclude),
		"digits":    removeCharacters(DigitCharacters, options.Exclude),
		"symbols":   removeCharacters(SymbolCharacters, options.Exclude),
	} {
		if !strings.ContainsAny(got, set) {
			t.Errorf("result has no %s character", name)
		}
	}
}

func removeCharacters(value, excluded string) string {
	return strings.Map(func(character rune) rune {
		if strings.ContainsRune(excluded, character) {
			return -1
		}
		return character
	}, value)
}

func TestRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name    string
		options Options
	}{
		{"nonpositive length", Options{Length: 0, Lowercase: true}},
		{"no class", Options{Length: 8}},
		{"too few positions", Options{Length: 2, Lowercase: true, Uppercase: true, Digits: true}},
		{"emptied class", Options{Length: 8, Digits: true, Exclude: DigitCharacters}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Generate("site", "login", "master", test.options); err == nil {
				t.Fatal("Generate() succeeded for invalid options")
			}
		})
	}
}
