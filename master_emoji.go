package pass

import "crypto/sha256"

// masterEmojiTable is the fixed alphabet for the master-password fingerprint.
// Entries are single-codepoint emoji with distinct silhouettes so a wrong
// master password is obvious at a glance. The order is part of the public
// derivation: never reorder or extend without treating it as a format change.
var masterEmojiTable = []rune{
	// animals
	'🐶', '🐱', '🐭', '🐹', '🐰', '🦊', '🐻', '🐼',
	'🐨', '🐯', '🦁', '🐮', '🐷', '🐸', '🐵', '🐔',
	'🐧', '🦉', '🐺', '🐴', '🦄', '🐝', '🦋', '🐢',
	// food
	'🍎', '🍊', '🍋', '🍉', '🍇', '🍓', '🍒', '🍑',
	'🥑', '🌽', '🍕', '🍔', '🍟', '🍩',
	// nature
	'🌵', '🌲', '🌳', '🌴', '🌱', '🌻', '🌸', '🌈',
	'⭐', '🌙',
	// objects
	'🔥', '💧', '⛄', '🎉', '🎸', '🎯', '🎲', '🎁',
	'🚀', '🚗', '⚓', '🎨', '🔑', '💡', '📚', '🎧',
}

// MasterEmojiCount is the number of emoji shown for one master password.
const MasterEmojiCount = 4

// MasterPasswordEmoji derives a short emoji fingerprint from a master
// password: the first bytes of its SHA-256 select MasterEmojiCount emoji from
// masterEmojiTable. The same master password always produces the same emoji,
// so a user can recognise a correctly typed master password across sessions
// without the password ever being stored or displayed. The fingerprint leaks
// only 24 bits of the hash, which does not help recover a strong master.
func MasterPasswordEmoji(master string) []rune {
	sum := sha256.Sum256([]byte(master))
	emoji := make([]rune, MasterEmojiCount)
	for i := range emoji {
		emoji[i] = masterEmojiTable[int(sum[i])%len(masterEmojiTable)]
	}
	return emoji
}

// MasterPasswordEmojiString is MasterPasswordEmoji as a display string.
func MasterPasswordEmojiString(master string) string {
	return string(MasterPasswordEmoji(master))
}

// MasterEmojiCodepoints lists the table's codepoints. Registration with a
// font subset needs exactly these glyphs.
func MasterEmojiCodepoints() []rune {
	codepoints := make([]rune, len(masterEmojiTable))
	copy(codepoints, masterEmojiTable)
	return codepoints
}
