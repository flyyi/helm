/*
Copyright The Helm Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package getter

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"k8s.io/helm/pkg/cli"
	"k8s.io/helm/pkg/helmpath"
)

func hh(debug bool) cli.EnvSettings {
	apath, err := filepath.Abs("./testdata")
	if err != nil {
		panic(err)
	}
	hp := helmpath.Home(apath)
	return cli.EnvSettings{
		Home:  hp,
		Debug: debug,
	}
}

func TestCollectPlugins(t *testing.T) {
	// Reset HELM HOME to testdata.
	oldhh := os.Getenv("HELM_HOME")
	defer os.Setenv("HELM_HOME", oldhh)
	os.Setenv("HELM_HOME", "")

	env := hh(false)
	p, err := collectPlugins(env)
	if err != nil {
		t.Fatal(err)
	}

	if len(p) != 2 {
		t.Errorf("Expected 2 plugins, got %d: %v", len(p), p)
	}

	if _, err := p.ByScheme("test2"); err != nil {
		t.Error(err)
	}

	if _, err := p.ByScheme("test"); err != nil {
		t.Error(err)
	}

	if _, err := p.ByScheme("nosuchthing"); err == nil {
		t.Fatal("did not expect protocol handler for nosuchthing")
	}
}

func TestPluginGetter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO: refactor this test to work on windows")
	}

	oldhh := os.Getenv("HELM_HOME")
	defer os.Setenv("HELM_HOME", oldhh)
	os.Setenv("HELM_HOME", "")

	env := hh(false)
	pg := newPluginGetter("echo", env, "test", ".")
	g, err := pg("test://foo/bar", "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	data, err := g.Get("test://foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	expect := "test://foo/bar"
	got := strings.TrimSpace(data.String())
	if got != expect {
		t.Errorf("Expected %q, got %q", expect, got)
	}
}
