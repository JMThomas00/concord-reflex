//go:build ignore

// release builds Reflex for every OS/CPU a Concord server runs on and packs
// each into dist/concord-reflex_<os>_<arch>.zip, the layout Concord's
// installer looks for:
//
//	concord-reflex/concord-reflex(.exe)    the server half
//	concord-reflex/plugin.toml             with the version stamped in
//	concord-reflex/client/plugin.wasm      the game, run in players' clients
//	concord-reflex/client/plugin.wasm.sig  its signature
//
// The client code is signed with the publisher key, which never leaves the
// publisher's computer: $CONCORD_PUBLISHER_KEY (the key file's contents),
// else the file named by $CONCORD_PUBLISHER_KEY_FILE, else publisher.key.
//
//	go run release.go            # version from the latest git tag
//	VERSION=1.2.0 go run release.go
package main

import (
	"archive/zip"
	"crypto/ed25519"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JMThomas00/Concord/sdk/codesign"
)

var targets = [][2]string{
	{"linux", "amd64"}, {"linux", "arm64"},
	{"windows", "amd64"},
	{"darwin", "amd64"}, {"darwin", "arm64"},
}

func main() {
	version := strings.TrimPrefix(os.Getenv("VERSION"), "v")
	if version == "" {
		if out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output(); err == nil {
			version = strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
		}
	}
	manifest, err := os.ReadFile("plugin.toml")
	check(err)
	if version != "" {
		manifest = regexp.MustCompile(`(?m)^version = "[^"]*"`).ReplaceAll(manifest, []byte(`version = "`+version+`"`))
	}

	// The client code: built once (WebAssembly runs everywhere) and signed.
	priv := publisherKey()
	public := codesign.FormatPublicKey(priv.Public().(ed25519.PublicKey))
	if m := regexp.MustCompile(`(?m)^publisher_key = "([^"]*)"`).FindSubmatch(manifest); m == nil || string(m[1]) != public {
		fail("plugin.toml's publisher_key isn't this key's (%s); clients would refuse the code", public)
	}
	wasm := filepath.Join("dist", "build", "plugin.wasm")
	build(wasm, "./clientcode", "GOOS=wasip1", "GOARCH=wasm")
	module, err := os.ReadFile(wasm)
	check(err)
	sig := codesign.Sign(priv, module)

	for _, t := range targets {
		goos, goarch := t[0], t[1]
		bin := "concord-reflex"
		if goos == "windows" {
			bin += ".exe"
		}
		out := filepath.Join("dist", "build", goos+"_"+goarch, bin)
		// CGO off: a static binary runs on any Linux, including Concord's
		// Alpine Docker image.
		build(out, ".", "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")

		zipPath := filepath.Join("dist", fmt.Sprintf("concord-reflex_%s_%s.zip", goos, goarch))
		f, err := os.Create(zipPath)
		check(err)
		zw := zip.NewWriter(f)
		add := func(name string, data []byte, mode os.FileMode) {
			h := &zip.FileHeader{Name: "concord-reflex/" + name, Method: zip.Deflate}
			h.SetMode(mode)
			w, err := zw.CreateHeader(h)
			check(err)
			_, err = w.Write(data)
			check(err)
		}
		binData, err := os.ReadFile(out)
		check(err)
		add(bin, binData, 0o755)
		add("plugin.toml", manifest, 0o644)
		add("client/plugin.wasm", module, 0o644)
		add("client/plugin.wasm.sig", sig, 0o644)
		check(zw.Close())
		check(f.Close())
		fmt.Println("packed", zipPath)
	}
	_ = os.RemoveAll(filepath.Join("dist", "build"))
}

func publisherKey() ed25519.PrivateKey {
	data := []byte(os.Getenv("CONCORD_PUBLISHER_KEY"))
	if len(data) == 0 {
		path := os.Getenv("CONCORD_PUBLISHER_KEY_FILE")
		if path == "" {
			path = "publisher.key"
		}
		var err error
		if data, err = os.ReadFile(path); err != nil {
			fail("no publisher key (%v): set CONCORD_PUBLISHER_KEY or CONCORD_PUBLISHER_KEY_FILE", err)
		}
	}
	k, err := codesign.ParsePrivateKey(data)
	check(err)
	return k
}

func build(out, pkg string, env ...string) {
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", out, pkg)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	check(cmd.Run())
}

func check(err error) {
	if err != nil {
		fail("%v", err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "release: "+format+"\n", args...)
	os.Exit(1)
}
