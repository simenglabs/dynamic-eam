package httpx

import "testing"

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{"=1+1": "'=1+1", "+x": "'+x", "-x": "'-x", "@x": "'@x", "\tx": "'\tx", "plain": "plain", "": ""}
	for in, want := range cases {
		if got := CSVSafe(in); got != want {
			t.Errorf("CSVSafe(%q) = %q, want %q", in, got, want)
		}
	}
}
