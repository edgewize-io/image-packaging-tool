package imageref

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	hostPartS   = `(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?)`
	hostPortS   = `(?:` + hostPartS + `(?:` + regexp.QuoteMeta(`.`) + hostPartS + `)*` + regexp.QuoteMeta(`.`) + `?` + regexp.QuoteMeta(`:`) + `[0-9]+)`
	hostDomainS = `(?:` + hostPartS + `(?:(?:` + regexp.QuoteMeta(`.`) + hostPartS + `)+` + regexp.QuoteMeta(`.`) + `?|` + regexp.QuoteMeta(`.`) + `))`
	hostUpperS  = `(?:[a-zA-Z0-9]*[A-Z][a-zA-Z0-9-]*[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[A-Z][a-zA-Z0-9]*)`
	registryS   = `(?:` + hostDomainS + `|` + hostPortS + `|` + hostUpperS + `|localhost(?:` + regexp.QuoteMeta(`:`) + `[0-9]+)?)`
	repoPartS   = `[a-z0-9]+(?:(?:\.|_|__|-+)[a-z0-9]+)*`
	pathS       = `[/a-zA-Z0-9_\-. ~]+`
	tagS        = `[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}`
	digestS     = `[A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*[:][[:xdigit:]]{32,}`
	schemeRE    = regexp.MustCompile(`^([a-z]+)://(.+)$`)
	registryRE  = regexp.MustCompile(`^(` + registryS + `)$`)
	refRE       = regexp.MustCompile(`^(?:(` + registryS + `)` + regexp.QuoteMeta(`/`) + `)?` +
		`(` + repoPartS + `(?:` + regexp.QuoteMeta(`/`) + repoPartS + `)*)` +
		`(?:` + regexp.QuoteMeta(`:`) + `(` + tagS + `))?` +
		`(?:` + regexp.QuoteMeta(`@`) + `(` + digestS + `))?$`)
)

const (
	dockerLibrary = "library"
	// dockerRegistry is the name resolved in docker images on Hub.
	dockerRegistry = "docker.io"
	// dockerRegistryLegacy is the name resolved in docker images on Hub.
	dockerRegistryLegacy = "index.docker.io"
	// dockerRegistryDNS is the host to connect to for Hub.
	dockerRegistryDNS = "registry-1.docker.io"
)

type ImageRef struct {
	Registry   string // Registry is the server for the "reg" scheme.
	Repository string // Repository is the path on the registry for the "reg" scheme.
	Tag        string // Tag is a mutable tag for a reference.
	Digest     string // Digest is an immutable hash for a reference.
}

func NewImageRef(parse string) (ret ImageRef, err error) {
	matchRef := refRE.FindStringSubmatch(parse)
	if matchRef == nil || len(matchRef) < 5 {
		if refRE.FindStringSubmatch(strings.ToLower(parse)) != nil {
			return ImageRef{}, fmt.Errorf("invalid reference \"%s\", repo must be lowercase", parse)
		}
		return ImageRef{}, fmt.Errorf("invalid reference \"%s\"", parse)
	}

	ret.Registry = matchRef[1]
	ret.Repository = matchRef[2]
	ret.Tag = matchRef[3]
	ret.Digest = matchRef[4]

	// handle localhost use case since it matches the regex for a repo path entry
	repoPath := strings.Split(ret.Repository, "/")
	if ret.Registry == "" && repoPath[0] == "localhost" {
		ret.Registry = repoPath[0]
		ret.Repository = strings.Join(repoPath[1:], "/")
	}
	switch ret.Registry {
	case "", dockerRegistryDNS, dockerRegistryLegacy:
		ret.Registry = dockerRegistry
	}
	if ret.Registry == dockerRegistry && !strings.Contains(ret.Repository, "/") {
		ret.Repository = dockerLibrary + "/" + ret.Repository
	}
	if ret.Tag == "" && ret.Digest == "" {
		ret.Tag = "latest"
	}
	if ret.Repository == "" {
		err = fmt.Errorf("invalid reference \"%s\"", parse)
		return
	}

	return
}

func NewHost(registry string) (ImageRef, error) {
	ret := ImageRef{}
	matchReg := registryRE.FindStringSubmatch(registry)
	if matchReg == nil || len(matchReg) < 2 {
		return ImageRef{}, fmt.Errorf("parsing failed \"%s\"", registry)
	}
	ret.Registry = matchReg[1]
	if ret.Registry == "" {
		return ImageRef{}, fmt.Errorf("parsing failed \"%s\"", registry)
	}

	return ret, nil
}
