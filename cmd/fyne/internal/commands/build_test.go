package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BuildWasmVersion(t *testing.T) {
	expected := []mockRunner{
		{
			expectedValue: expectedValue{
				args:  []string{"build"},
				env:   []string{"GOARCH=wasm", "GOOS=js", "CGO_ENABLED=0"},
				osEnv: true,
				dir:   "myTest",
			},
			mockReturn: mockReturn{ret: []byte("")},
		},
	}

	wasmBuildTest := &testCommandRuns{runs: expected, t: t}
	b := &Builder{appData: &appData{}, os: "wasm", srcdir: "myTest", runner: wasmBuildTest}
	err := b.build()
	assert.NoError(t, err)
	wasmBuildTest.verifyExpectation()
}

func Test_BuildWasmReleaseVersion(t *testing.T) {
	expected := []mockRunner{
		{
			expectedValue: expectedValue{
				args:  []string{"build", "-trimpath", "-ldflags", "-s -w", "-tags", "release"},
				env:   []string{"GOARCH=wasm", "GOOS=js", "CGO_ENABLED=0"},
				osEnv: true,
				dir:   "myTest",
			},
			mockReturn: mockReturn{
				ret: []byte(""),
			},
		},
	}

	wasmBuildTest := &testCommandRuns{runs: expected, t: t}
	b := &Builder{appData: &appData{}, os: "wasm", srcdir: "myTest", release: true, runner: wasmBuildTest}
	err := b.build()
	assert.Nil(t, err)
	wasmBuildTest.verifyExpectation()
}

func Test_BuildLinuxReleaseVersion(t *testing.T) {
	relativePath := "." + string(os.PathSeparator) + filepath.Join("cmd", "terminal")

	cflags, exists := os.LookupEnv("CGO_CFLAGS")
	if exists {
		cflags += " "
	}
	ldflags, exists := os.LookupEnv("CGO_LDFLAGS")
	if exists {
		ldflags += " "
	}

	archcflags := ""
	arch := targetArch()
	switch arch {
	case "arm64":
		archcflags = " -mbranch-protection=bti+pac-ret"
	}

	hardeningCFlags := hardeningCFlagsLookup(ccVersion(), "linux", arch)
	expected := []mockRunner{
		{
			expectedValue: expectedValue{
				args:  []string{"build", "-trimpath", "-ldflags", "-s -w", "-tags", "release", relativePath},
				env:   []string{"CGO_ENABLED=1", "GOOS=linux", fmt.Sprintf("CGO_CFLAGS=%s%s %s%s", cflags, baseCFLAGSRelease, hardeningCFlags, archcflags), fmt.Sprintf("CGO_LDFLAGS=%s%s", ldflags, hardeningLDFLAGSLinux)},
				osEnv: true,
				dir:   "myTest",
			},
			mockReturn: mockReturn{
				ret: []byte(""),
			},
		},
	}

	linuxBuildTest := &testCommandRuns{runs: expected, t: t}
	b := &Builder{appData: &appData{}, os: "linux", srcdir: "myTest", release: true, runner: linuxBuildTest, goPackage: relativePath}
	err := b.build()
	assert.NoError(t, err)
	linuxBuildTest.verifyExpectation()
}

func Test_AppendEnv(t *testing.T) {
	env := []string{
		"foo=bar",
		"bar=baz",
		"foo1=bar=baz",
	}

	appendEnv(&env, "foo2", "baz2")
	appendEnv(&env, "foo", "baz")
	appendEnv(&env, "foo1", "-bar")

	if assert.Len(t, env, 4) {
		assert.Equal(t, "foo=bar baz", env[0])
		assert.Equal(t, "bar=baz", env[1])
		assert.Equal(t, "foo1=bar=baz -bar", env[2])
		assert.Equal(t, "foo2=baz2", env[3])
	}
}

type extractTest struct {
	value       string
	wantLdFlags string
}

func Test_ExtractLdFlags(t *testing.T) {
	goFlagsTests := []extractTest{
		{"-ldflags=-w", "-w"},
		{"-ldflags=-s", "-s"},
		{"-ldflags=-w -ldflags=-s", "-w -s"},
		{"-mod=vendor", ""},
		{"", ""},

		// entries are merged, so every -X applies
		{"-ldflags=-X=main.a=1 -ldflags=-X=main.b=2", "-X=main.a=1 -X=main.b=2"},
		{"-mod=vendor -ldflags=-X=main.a=1 -trimpath", "-X=main.a=1"},
		{"-ldflags=-w\t-mod=vendor\n-x", "-w"},
		{"-ldflags= -mod=vendor", ""},

		// double dash spelling counts too
		{"--ldflags=-X=main.a=1", "-X=main.a=1"},
		{"--ldflags=-w -ldflags=-s", "-w -s"},
		{"--mod=vendor", ""},

		// quoted entries keep their spaces
		{"'-ldflags=-X main.version=1.2.3'", "-X main.version=1.2.3"},
		{"'-ldflags=-X main.a=1' -ldflags=-X=main.b=2", "-X main.a=1 -X=main.b=2"},
		{`"-ldflags=-X main.a=1"`, "-X main.a=1"},
		{"'-gcflags=all=-N -l' -ldflags=-w", "-w"},

		// unparsable value yields nothing
		{"'-ldflags=-w", ""},
		{`"-ldflags=-w`, ""},
	}

	for _, test := range goFlagsTests {
		assert.Equal(t, test.wantLdFlags, extractLdFlags(test.value), "ldflags of %q", test.value)
	}
}

func Test_LdflagsFromGoFlags(t *testing.T) {
	goFlags := "'-ldflags=-X main.version=1.2.3' -mod=vendor"
	t.Setenv("GOFLAGS", goFlags)

	// the windows packager builds twice in one process, so both must see the flags
	assert.Equal(t, "-X main.version=1.2.3", ldflagsFromGoFlags())
	assert.Equal(t, "-X main.version=1.2.3", ldflagsFromGoFlags())
	assert.Equal(t, goFlags, os.Getenv("GOFLAGS"))
}

func Test_NormaliseVersion(t *testing.T) {
	assert.Equal(t, "master", normaliseVersion("master"))
	assert.Equal(t, "2.3.0.0", normaliseVersion("v2.3"))
	assert.Equal(t, "2.4.0.0", normaliseVersion("v2.4.0"))
	assert.Equal(t, "2.3.6.0-dev", normaliseVersion("v2.3.6-0.20230711180435-d4b95e1cb1eb"))
	assert.Equal(t, "2.4.1.0-dev", normaliseVersion("v2.4.1-rc7.0.20230711180435-d4b95e1cb1eb"))
}
