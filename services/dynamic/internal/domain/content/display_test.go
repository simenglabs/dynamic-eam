package content

import (
	"testing"

	"eam/services/common/errs"
)

func TestDisplayFieldAcceptsNameOrTemplate(t *testing.T) {
	for _, d := range []string{"plate", "{plate}", "{plate} - {status}", "Kendaraan {plate} ({since})", "{location}"} {
		ct := vehicle()
		ct.DisplayField = d
		if err := ct.Validate(catalog()); err != nil {
			t.Errorf("%q: %v", d, err)
		}
	}
}

func TestDisplayFieldRejects(t *testing.T) {
	for _, d := range []string{"", "nope", "{nope}", "{plate} {addresses}", "no placeholder", "{plate"} {
		ct := vehicle()
		ct.DisplayField = d
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%q must be invalid", d)
		}
	}
}

func TestDisplayLabel(t *testing.T) {
	ct := vehicle()
	data := map[string]any{"plate": "L 1", "status": "Aktif", "year": 2021.0}
	cases := map[string]string{
		"plate":              "L 1",
		"{plate} - {status}": "L 1 - Aktif",
		"{plate} ({since})":  "L 1 ()", // empty field = empty string
		"Kendaraan #{plate}": "Kendaraan #L 1",
	}
	for tpl, want := range cases {
		ct.DisplayField = tpl
		if got := ct.DisplayLabel(data); got != want {
			t.Errorf("%q: %q, want %q", tpl, got, want)
		}
	}
}

func TestDisplaySortField(t *testing.T) {
	ct := vehicle()
	for tpl, want := range map[string]string{"plate": "plate", "{status} {plate}": "status", "#{plate}": "plate"} {
		ct.DisplayField = tpl
		if got := ct.DisplaySortField(); got != want {
			t.Errorf("%q: %q, want %q", tpl, got, want)
		}
	}
}
