package fs

import (
	"strconv"
)

func strconvFormatFloat(v float64) string {
	// Compact representation: integers without decimal point, otherwise 2 decimals trimmed.
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}
