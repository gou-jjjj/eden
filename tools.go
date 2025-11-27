package eden

import (
	"crypto/md5"
	"encoding/hex"
)

func FillMap(src ...[]string) map[string]string {
	m := map[string]string{}
	if len(src) == 0 {
		return m
	}

	key := src[0]
	val := src[0]
	if len(src) > 1 {
		val = src[1]
	}
	if len(key) != len(val) {
		return m
	}

	for i, k := range key {
		m[k] = val[i]
	}
	return m
}

func CombineMap(maps ...map[string]string) map[string]string {
	m := map[string]string{}
	for _, mm := range maps {
		for k, v := range mm {
			m[k] = v
		}
	}
	return m
}

func Md5(token string) string {
	m := md5.New()
	m.Write([]byte(token))
	return hex.EncodeToString(m.Sum(nil))
}
