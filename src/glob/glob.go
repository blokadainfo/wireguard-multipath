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
	var re strings.Builder
	re.WriteString("^")

	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '*':
			re.WriteString(".*")
		case '?':
			re.WriteString(".")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '\\':
			re.WriteByte('\\')
			re.WriteByte(expr[i])
		default:
			re.WriteByte(expr[i])
		}
	}

	re.WriteString("$")
	regexp, err := regexp.Compile(re.String())

	return Glob{regexp: regexp}, err
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
