package envutils

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/glob"
)

func get[T bool | int64 | time.Duration | float64 | string | []string | *regexp.Regexp | []*regexp.Regexp | glob.Glob | []glob.Glob](conv func(string) (T, error), key string, fallback ...T) T {
	value, ok := os.LookupEnv(key)
	if ok {
		if v, err := conv(value); err != nil {
			log.Fatalf("Environment variable couldn't be converted to the proper type: %v: %v", key, value)
		} else {
			return v
		}
	}

	if len(fallback) == 0 {
		log.Fatalf("Environment variable must be set: %v", key)
	}

	if len(fallback) > 1 {
		panic("only 1 fallback parameter can be passed")
	}

	return fallback[0]
}

func GetGlobList(key string, fallback ...[]glob.Glob) []glob.Glob {
	return get(func(str string) ([]glob.Glob, error) {
		strs := strings.Split(str, ",")
		globs := make([]glob.Glob, 0, len(strs))
		for _, s := range strs {
			g, err := glob.Compile(s)
			if err != nil {
				return globs, fmt.Errorf("%s", "regexp: Compile("+quote(str)+"): "+err.Error())
			}

			globs = append(globs, g)
		}
		return globs, nil
	}, key, fallback...)
}

func GetGlob(key string, fallback ...glob.Glob) glob.Glob {
	return get(func(str string) (glob.Glob, error) {
		return glob.Compile(str)
	}, key, fallback...)
}

func GetRegexpList(key string, fallback ...[]*regexp.Regexp) []*regexp.Regexp {
	return get(func(str string) ([]*regexp.Regexp, error) {
		strs := strings.Split(str, ",")
		regexps := make([]*regexp.Regexp, 0, len(strs))
		for _, s := range strs {
			r, err := regexp.Compile(s)
			if err != nil {
				return regexps, fmt.Errorf("%s", "regexp: Compile("+quote(str)+"): "+err.Error())
			}

			regexps = append(regexps, r)
		}
		return regexps, nil
	}, key, fallback...)
}

func GetRegexp(key string, fallback ...*regexp.Regexp) *regexp.Regexp {
	return get(func(str string) (*regexp.Regexp, error) {
		return regexp.Compile(str)
	}, key, fallback...)
}

func GetStringList(key string, fallback ...[]string) []string {
	return get(func(str string) ([]string, error) {
		return strings.Split(str, ","), nil
	}, key, fallback...)
}

func GetString(key string, fallback ...string) string {
	return get(func(str string) (string, error) {
		return str, nil
	}, key, fallback...)
}

func GetFloat(key string, fallback ...float64) float64 {
	return get(func(str string) (float64, error) {
		return strconv.ParseFloat(str, 64)
	}, key, fallback...)
}

func GetTimeDuration(quantifier time.Duration, key string, fallback ...time.Duration) time.Duration {
	return get(func(str string) (time.Duration, error) {
		i, err := strconv.ParseInt(str, 10, 64)
		return time.Duration(i) * quantifier, err
	}, key, fallback...) * quantifier
}

func GetInt(key string, fallback ...int64) int64 {
	return get(func(str string) (int64, error) {
		return strconv.ParseInt(str, 10, 64)
	}, key, fallback...)
}

func GetBool(key string, fallback ...bool) bool {
	return get(func(str string) (bool, error) {
		return strconv.ParseBool(str)
	}, key, fallback...)
}

func quote(s string) string {
	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}
