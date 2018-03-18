package base62

import (
    "math"
)

const (
    alphabet  = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
    base = 62
)

var m = make(map[rune]int)

func init() {
    for idx, char := range alphabet {
        m[char] = idx
    }
}

func Encode(num int) string {
    if num == 0 {
        return "0"
    }
    var s []byte
    for num > 0 {
        rem := num % base
        num /= base
        s = append(s, alphabet[rem])
    }
    for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
        s[i], s[j] = s[j], s[i]
    }
    return string(s)
}

func Decode(str string) (ret int) {
    strlen := len(str)
    for idx, char := range str {
        ret += m[char] * int(math.Pow(base, float64(strlen - idx - 1)))
    }
    return
}