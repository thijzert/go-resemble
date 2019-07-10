package resemble

// Resemble checks if integers a and b resemble each other.  It is included in
// this package so that it has at least one exported function, which makes it
// importable, which enables one to vendor it using `go mod`.
func Resemble(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	if a < 0 {
		a *= -1
	}
	if b < 0 {
		b *= -1
	}
	if a < b {
		a, b = b, a
	}

	for b > 1 {
		r := a % b
		if r == 0 {
			return b
		}
		a, b = b, r
	}
	return 1
}
