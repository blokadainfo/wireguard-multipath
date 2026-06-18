package envutils

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func get[T bool | int64 | time.Duration | float64 | string | []string](conv func(string) (T, error), key string, fallback ...T) T {
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
