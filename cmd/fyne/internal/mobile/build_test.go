package mobile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LdflagsFor(t *testing.T) {
	tests := []struct {
		ldflags    string
		stripDWARF bool
		want       string
	}{
		{"", false, ""},
		{"", true, "-w"},
		{"-X=main.a=1", false, "-X=main.a=1 -s=false"},
		{"-X=main.a=1", true, "-X=main.a=1 -s=false -w"},
		{"-X main.a=1 -X main.b=2", true, "-X main.a=1 -X main.b=2 -s=false -w"},

		// -s is overridden, not removed, so a -s meant for another flag survives
		{"-s", true, "-s -s=false -w"},
		{"-s -w", false, "-s -w -s=false"},
		{"-extldflags -s", true, "-extldflags -s -s=false -w"},
	}

	for _, test := range tests {
		assert.Equal(t, test.want, ldflagsFor(test.ldflags, test.stripDWARF),
			"ldflagsFor(%q, %v)", test.ldflags, test.stripDWARF)
	}
}
