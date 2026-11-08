package envutils

import (
	"os"
	"testing"
	"time"
)

// func TestGetGlobList(t *testing.T) {
// }

// func TestGetGlob(t *testing.T) {
// }

// func TestGetRegexpList(t *testing.T) {
// }

// func TestGetRegexp(t *testing.T) {
// }

// func TestGetCIDRList(t *testing.T) {
// }

// func TestGetCIDR(t *testing.T) {
// }

// func TestGetStringList(t *testing.T) {
// }

// func TestGetString(t *testing.T) {
// }

// func TestGetFloat(t *testing.T) {
// }

func TestGetTimeDuration(t *testing.T) {
	{
		os.Setenv("TEST_GET_TD_1", "10")
		got := GetTimeDuration(time.Millisecond, "TEST_GET_TD_1")
		want := 10 * time.Millisecond
		if got != want {
			t.Errorf("GetTimeDuration(time.Millisecond, \"10\") = %v, want %v", got, want)
		}
	}

	{
		os.Setenv("TEST_GET_TD_2", "123")
		got := GetTimeDuration(time.Second, "TEST_GET_TD_2")
		want := 123 * time.Second
		if got != want {
			t.Errorf("GetTimeDuration(time.Second, \"123\") = %v, want %v", got, want)
		}
	}

	{
		// os.Setenv("TEST_GET_TD_3", "")
		got := GetTimeDuration(time.Second, "TEST_GET_TD_3", 1000*time.Millisecond)
		want := 1000 * time.Millisecond
		if got != want {
			t.Errorf("GetTimeDuration(time.Second, \"\", 1000*time.Millisecond) = %v, want %v", got, want)
		}
	}
}

// func TestGetInt(t *testing.T) {
// }

// func TestGetBool(t *testing.T) {
// }
