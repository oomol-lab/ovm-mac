package dism

import (
	"errors"
	"testing"
)

func TestDismDisableFeature(t *testing.T) {
	testFeatureList := []struct {
		name     string
		expected error
	}{
		{"TFTP", nil},
		{"TelnetClient", nil},
	}

	for _, test := range testFeatureList {
		t.Run(test.name, func(t *testing.T) {

			dismSession, err := OpenSession(DISM_ONLINE_IMAGE,
				"",
				"",
				DismLogErrorsWarningsInfo,
				"",
				"")

			if err != nil {
				panic(err)
			}
			defer dismSession.Close()

			result := dismSession.DisableFeature(test.name, "", false, nil, nil)
			if !errors.Is(result, test.expected) {
				t.Errorf("DismDisableFeature() expected %d, got %d", test.expected, result)
			}
		})
	}
}
