package mask

import (
	"regexp"
	"strings"
)

var autoName = regexp.MustCompile(`(?i)(PASSWORD|SECRET|TOKEN|KEY)`)

func IsSecretKey(key string, declared []string) bool {
	for _, d := range declared {
		if d == key {
			return true
		}
	}
	return autoName.MatchString(key)
}

func ShouldMaskValue(val string, minLen int) bool {
	if minLen <= 0 {
		minLen = 4
	}
	return len(val) >= minLen
}

func MaskMap(values map[string]string, declaredSecrets []string, minLen int) map[string]string {
	out := make(map[string]string, len(values))
	for k, v := range values {
		if IsSecretKey(k, declaredSecrets) && ShouldMaskValue(v, minLen) {
			out[k] = "***"
		} else {
			out[k] = v
		}
	}
	return out
}

// Apply replaces secret values in text (longest first to reduce partial corruption).
func Apply(text string, secrets []string, minLen int) string {
	type pair struct{ v string }
	var vals []string
	for _, s := range secrets {
		if ShouldMaskValue(s, minLen) {
			vals = append(vals, s)
		}
	}
	// naive longest-first
	for i := 0; i < len(vals); i++ {
		for j := i + 1; j < len(vals); j++ {
			if len(vals[j]) > len(vals[i]) {
				vals[i], vals[j] = vals[j], vals[i]
			}
		}
	}
	out := text
	for _, v := range vals {
		out = strings.ReplaceAll(out, v, "***")
	}
	return out
}
