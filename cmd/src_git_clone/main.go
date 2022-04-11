package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shanexu/sillytools/common"
)

var (
	// scpSyntax was modified from https://golang.org/src/cmd/go/vcs.go.
	scpSyntax = regexp.MustCompile(`^([a-zA-Z0-9_]+@)?([a-zA-Z0-9._-]+):(.*)$`)

	// Transports is a set of known Git URL schemes.
	Transports = NewTransportSet(
		"ssh",
		"git",
		"git+ssh",
		"http",
		"https",
		"ftp",
		"ftps",
		"rsync",
		"file",
	)
)

// Parser converts a string into a URL.
type Parser func(string) (*url.URL, error)

// Parse parses rawurl into a URL structure. Parse first attempts to
// find a standard URL with a valid Git transport as its scheme. If
// that cannot be found, it then attempts to find a SCP-like URL. And
// if that cannot be found, it assumes rawurl is a local path. If none
// of these rules apply, Parse returns an error.
func Parse(rawurl string) (u *url.URL, err error) {
	parsers := []Parser{
		ParseTransport,
		ParseScp,
		ParseLocal,
	}

	// Apply each parser in turn; if the parser succeeds, accept its
	// result and return.
	for _, p := range parsers {
		u, err = p(rawurl)
		if err == nil {
			return u, err
		}
	}

	// It's unlikely that none of the parsers will succeed, since
	// ParseLocal is very forgiving.
	return new(url.URL), fmt.Errorf("failed to parse %q", rawurl)
}

// ParseTransport parses rawurl into a URL object. Unless the URL's
// scheme is a known Git transport, ParseTransport returns an error.
func ParseTransport(rawurl string) (*url.URL, error) {
	u, err := url.Parse(rawurl)
	if err == nil && !Transports.Valid(u.Scheme) {
		err = fmt.Errorf("scheme %q is not a valid transport", u.Scheme)
	}
	if u != nil && u.User == nil {
		u.User = url.User("")
	}
	return u, err
}

// ParseScp parses rawurl into a URL object. The rawurl must be
// an SCP-like URL, otherwise ParseScp returns an error.
func ParseScp(rawurl string) (*url.URL, error) {
	match := scpSyntax.FindAllStringSubmatch(rawurl, -1)
	if len(match) == 0 {
		return nil, fmt.Errorf("no scp URL found in %q", rawurl)
	}
	m := match[0]
	return &url.URL{
		Scheme: "ssh",
		User:   url.User(strings.TrimRight(m[1], "@")),
		Host:   m[2],
		Path:   m[3],
	}, nil
}

// ParseLocal parses rawurl into a URL object with a "file"
// scheme. This will effectively never return an error.
func ParseLocal(rawurl string) (*url.URL, error) {
	return &url.URL{
		Scheme: "file",
		User:   url.User(""),
		Host:   "",
		Path:   rawurl,
	}, nil
}

// TransportSet represents a set of valid Git transport schemes. It
// maps these schemes to empty structs, providing a set-like
// interface.
type TransportSet struct {
	Transports map[string]struct{}
}

// NewTransportSet returns a TransportSet with the items keys mapped
// to empty struct values.
func NewTransportSet(items ...string) *TransportSet {
	t := &TransportSet{
		Transports: map[string]struct{}{},
	}
	for _, i := range items {
		t.Transports[i] = struct{}{}
	}
	return t
}

// Valid returns true if transport is a known Git URL scheme and false
// if not.
func (t *TransportSet) Valid(transport string) bool {
	_, ok := t.Transports[transport]
	return ok
}

var ErrSrcNotFound = errors.New("src not found")

func findAncestorSrc() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir != "/" {
		name := filepath.Base(dir)
		if name == "src" {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", ErrSrcNotFound
}

func findEnvSrc() (string, error) {
	srcRoot := os.Getenv("SRC_ROOT")
	if srcRoot == "" {
		return "", ErrSrcNotFound
	}
	return srcRoot, nil
}

func findDefaultSrc() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, "src"), nil
}

var findSrc = common.Alternatives(findAncestorSrc, findEnvSrc, findDefaultSrc)

func main() {
	src, err := findSrc()
	if err != nil {
		panic(err)
	}
	if len(os.Args) <= 1 {
		_, _ = fmt.Fprintln(os.Stderr, "src_git_clone url")
		os.Exit(1)
	}
	source := os.Args[1]
	u, err := Parse(source)
	if err != nil {
		panic(fmt.Sprintf("url parse failed %s", err))
	}
	var parent string
	idx := strings.LastIndex(u.Path, "/")
	if idx > 0 {
		parent = u.Path[0:idx]
	}
	parents := append([]string{src, simpleHost(u.Host)}, strings.Split(parent, "/")...)
	path := filepath.Join(parents...)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			panic(err)
		}
	}
	parts := strings.Split(u.Path, "/")
	target := filepath.Join(append([]string{u.Host}, parts...)...)
	idx = strings.LastIndex(target, ".")
	if idx > 0 {
		target = target[0:idx]
	}
	cmd := exec.Command("git", "clone", source)
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func simpleHost(host string) string {
	idx := strings.Index(host, ":")
	if idx != -1 {
		host = host[0:idx]
	}
	return host
}
