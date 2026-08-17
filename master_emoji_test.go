package gopass

import "testing"

func TestMasterPasswordEmojiIsDeterministic(t *testing.T) {
	first := MasterPasswordEmojiString("correct horse battery staple")
	for i := 0; i < 8; i++ {
		if got := MasterPasswordEmojiString("correct horse battery staple"); got != first {
			t.Fatalf("fingerprint changed between calls: %q vs %q", got, first)
		}
	}
	if got := MasterPasswordEmojiString(""); got == "" || len([]rune(got)) != MasterEmojiCount {
		t.Fatalf("empty master fingerprint = %q, want %d emoji", got, MasterEmojiCount)
	}
}

func TestMasterPasswordEmojiDiffersForDifferentMasters(t *testing.T) {
	seen := map[string]string{
		MasterPasswordEmojiString("test"):                         "test",
		MasterPasswordEmojiString("test2"):                        "test2",
		MasterPasswordEmojiString("tset"):                         "tset",
		MasterPasswordEmojiString("hunter2"):                      "hunter2",
		MasterPasswordEmojiString("correct horse battery staple"): "correct horse battery staple",
	}
	// Five distinct short masters colliding on a 24-bit space would be
	// itself a sha256 anomaly; a single collision here means a table bug.
	if len(seen) != 5 {
		t.Fatalf("expected distinct fingerprints, got %d unique for 5 masters", len(seen))
	}
}

func TestMasterPasswordEmojiUsesTableAlphabet(t *testing.T) {
	inTable := func(r rune) bool {
		for _, candidate := range masterEmojiTable {
			if candidate == r {
				return true
			}
		}
		return false
	}
	if len(masterEmojiTable) != 64 {
		t.Fatalf("table has %d emoji, want 64", len(masterEmojiTable))
	}
	for _, master := range []string{"a", "b", "ab", "🔐", "фыв"} {
		for _, r := range MasterPasswordEmoji(master) {
			if !inTable(r) {
				t.Fatalf("master %q produced rune %U outside the table", master, r)
			}
		}
	}
}

func TestMasterEmojiCodepointsMatchTable(t *testing.T) {
	codepoints := MasterEmojiCodepoints()
	if len(codepoints) != len(masterEmojiTable) {
		t.Fatalf("got %d codepoints, want %d", len(codepoints), len(masterEmojiTable))
	}
	for i := range codepoints {
		if codepoints[i] != masterEmojiTable[i] {
			t.Fatalf("codepoint %d = %U, want %U", i, codepoints[i], masterEmojiTable[i])
		}
	}
}
