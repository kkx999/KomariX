package plugin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kkx999/KomariX/utils"
)

// CheckKomariXVersion validates a manifest KomariX compatibility constraint against the
// running server version. Supported constraints: empty (any version),
// "x.y.z" (exact), and ">=x.y.z", ">x.y.z", "<=x.y.z", "<x.y.z". An optional
// leading "v" is accepted.
func CheckKomariXVersion(constraint string) error {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		return nil
	}
	rest := constraint
	op := ""
	for _, candidate := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(rest, candidate) {
			op = candidate
			rest = strings.TrimSpace(strings.TrimPrefix(rest, candidate))
			break
		}
	}
	want, err := parseSemver(rest)
	if err != nil {
		return fmt.Errorf("invalid KomariX version constraint %q: %w", constraint, err)
	}

	// Native KomariX plugins may target the product version, while legacy Komari
	// plugins carry the upstream Komari version they were built against. Accept
	// the package when either the running product version or the explicitly
	// maintained plugin-API compatibility baseline satisfies the constraint.
	if have, err := parseSemver(utils.CurrentVersion); err == nil && satisfies(compareSemver(have, want), op) {
		return nil
	}
	if compat, err := parseSemver(utils.PluginCompatibilityVersion); err == nil && satisfies(compareSemver(compat, want), op) {
		return nil
	}

	return fmt.Errorf(
		"plugin requires KomariX/plugin API %s, running %s (plugin API compatibility %s)",
		constraint,
		utils.CurrentVersion,
		utils.PluginCompatibilityVersion,
	)
}

func satisfies(cmp int, op string) bool {
	switch op {
	case ">=":
		return cmp >= 0
	case ">":
		return cmp > 0
	case "<=":
		return cmp <= 0
	case "<":
		return cmp < 0
	default: // exact
		return cmp == 0
	}
}

func parseSemver(s string) ([3]int, error) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "v"))
	if s == "" {
		return [3]int{}, fmt.Errorf("empty version")
	}
	parts := strings.Split(s, ".")
	if len(parts) > 3 {
		return [3]int{}, fmt.Errorf("too many version parts")
	}
	var v [3]int
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return [3]int{}, fmt.Errorf("invalid version part %q", part)
		}
		v[i] = n
	}
	return v, nil
}

func compareSemver(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
