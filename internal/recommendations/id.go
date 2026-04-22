package recommendations

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashID returns a stable 16-character hex identifier for a recommendation
// derived from its target URL, target query, and change type. The same
// inputs always produce the same ID across runs, so re-running the
// pipeline upserts existing rows rather than duplicating them.
func HashID(targetURL, targetQuery string, changeType ChangeType) string {
	h := sha256.New()
	// Use NUL separators so fields can't collide via concatenation.
	h.Write([]byte(targetURL))
	h.Write([]byte{0})
	h.Write([]byte(targetQuery))
	h.Write([]byte{0})
	h.Write([]byte(changeType))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
