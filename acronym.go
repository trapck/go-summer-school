package main

import (
	"fmt"
	"strings"
)

func accr(s string, alg func(string) string) string {
	result := alg(s)
	return fmt.Sprintf("Input: %v; Accr: %v; Length: %v\n", s, result, len(result))
}

func loopAlg(s string) string {
	space, result := " ", ""
	for i, v := range s {
		next:= ""
		switch {
			case i == 0 && string(s[i+1]) != space:
				next= string(v)
			case string(v) == space && i + 1 < len(s) && string(s[i+1]) != space:
				next= string(s[i+1])
		}
		result+= strings.ToUpper(next)
	}
	return result
}

func splitAlg(s string) string {
	result := ""
	for _, v := range strings.Fields(s) {
		result+= strings.ToUpper(string(v[0]))
	}
	return result
}

func main() {
	input := "   hello  world  "
	algs := map[string]func(string) string {
		"loop": loopAlg,
		"split": splitAlg,
	}
	for name, alg := range algs {
		fmt.Printf("Alg: %v => %v\n", name, accr(input, alg))
	}
}
