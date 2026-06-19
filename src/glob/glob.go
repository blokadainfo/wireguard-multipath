package glob

import (
	"regexp"
	"strconv"
	"strings"
)

type Glob struct {
	regexp *regexp.Regexp
}

func Compile(expr string) (Glob, error) {
	var e strings.Builder
	e.WriteString("^")

	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '*':
			e.WriteString(".*")
		case '?':
			e.WriteString(".")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '\\':
			e.WriteByte('\\')
			e.WriteByte(expr[i])
		default:
			e.WriteByte(expr[i])
		}
	}

	e.WriteString("$")
	re, err := regexp.Compile(e.String())

	return Glob{regexp: re}, err
}

func MustCompile(str string) Glob {
	glob, err := Compile(str)
	if err != nil {
		panic(`glob: Compile(` + quote(str) + `): ` + err.Error())
	}

	return glob
}

func (g Glob) MatchString(s string) bool {
	return g.regexp.MatchString(s)
}

func (g Glob) Match(b []byte) bool {
	return g.regexp.Match(b)
}

func (g Glob) String() string {
	return g.regexp.String()
}

func quote(s string) string {
	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}
