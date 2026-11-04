package updater

import (
	"regexp"
	"strconv"
	"strings"
)

// Pre-release weights: larger means newer.
var preWeights = map[string]int{
	"dev":   1,
	"alpha": 2,
	"beta":  3,
	"rc":    4,
}

var versionPattern = regexp.MustCompile(`^(\d+(?:\.\d+)*)(?:[-_.]([a-zA-Z]+)(?:[._-]?(\d+))?)?$`)

type parsedVersion struct {
	nums   []int
	pre    string
	preNum int
	hasPre bool
}

// ParseVersion parses a semantic version such as "v0.8.0" or "0.8.0-beta.1".
// It returns false for tags like "2026-11-04" or "snapshot".
func ParseVersion(raw string) (parsedVersion, bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	m := versionPattern.FindStringSubmatch(s)
	if m == nil {
		return parsedVersion{}, false
	}
	parts := strings.Split(m[1], ".")
	nums := make([]int, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return parsedVersion{}, false
		}
		nums = append(nums, n)
	}
	for len(nums) < 3 {
		nums = append(nums, 0)
	}
	pv := parsedVersion{nums: nums}
	if m[2] != "" {
		pv.hasPre = true
		pv.pre = strings.ToLower(m[2])
		if m[3] != "" {
			pv.preNum, _ = strconv.Atoi(m[3])
		}
	}
	return pv, true
}

// CompareVersions returns 1 when a is newer than b, -1 when older, 0 when equal.
// A release version is newer than any of its pre-releases (0.8.0 > 0.8.0-beta).
// Non-semantic versions compare equal so callers can ignore them.
func CompareVersions(a, b string) int {
	pa, okA := ParseVersion(a)
	pb, okB := ParseVersion(b)
	if !okA || !okB {
		return 0
	}
	length := len(pa.nums)
	if len(pb.nums) > length {
		length = len(pb.nums)
	}
	for i := 0; i < length; i++ {
		var na, nb int
		if i < len(pa.nums) {
			na = pa.nums[i]
		}
		if i < len(pb.nums) {
			nb = pb.nums[i]
		}
		if na > nb {
			return 1
		}
		if na < nb {
			return -1
		}
	}
	if !pa.hasPre && !pb.hasPre {
		return 0
	}
	if !pa.hasPre {
		return 1
	}
	if !pb.hasPre {
		return -1
	}
	wa := preWeights[pa.pre]
	wb := preWeights[pb.pre]
	if wa > wb {
		return 1
	}
	if wa < wb {
		return -1
	}
	if pa.preNum > pb.preNum {
		return 1
	}
	if pa.preNum < pb.preNum {
		return -1
	}
	return 0
}

// IsSemanticVersion reports whether raw can be compared by CompareVersions.
func IsSemanticVersion(raw string) bool {
	_, ok := ParseVersion(raw)
	return ok
}
