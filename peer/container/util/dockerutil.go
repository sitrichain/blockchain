package util

import (
	"runtime"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/metadata"
)

//NewDockerClient creates a docker client
func NewDockerClient() (client *docker.Client, err error) {
	endpoint := conf.V.Peer.VM.Endpoint
	tlsenabled := conf.V.Peer.VM.TLS.Enabled
	if tlsenabled {
		cert := conf.V.Peer.VM.TLS.Certificate
		key := conf.V.Peer.VM.TLS.PrivateKey
		ca := conf.V.Peer.VM.TLS.RootCAs[0]
		client, err = docker.NewTLSClient(endpoint, cert, key, ca)
	} else {
		client, err = docker.NewClient(endpoint)
	}
	return
}

// Our docker images retrieve $ARCH via "uname -m", which is typically "x86_64" for, well, x86_64.
// However, GOARCH uses "amd64".  We therefore need to normalize any discrepancies between "uname -m"
// and GOARCH here.
var archRemap = map[string]string{
	"amd64": "x86_64",
}

func getArch() string {
	if remap, ok := archRemap[runtime.GOARCH]; ok {
		return remap
	} else {
		return runtime.GOARCH
	}
}

func ParseDockerfileTemplate(template string) string {
	r := strings.NewReplacer(
		"$(ARCH)", getArch(),
		"$(PROJECT_VERSION)", metadata.Version,
		"$(BASE_VERSION)", metadata.BaseVersion,
		"$(DOCKER_NS)", metadata.DockerNamespace,
		"$(BASE_DOCKER_NS)", metadata.BaseDockerNamespace,
		"$(GIT_VERSION)", metadata.GitVersion)

	return r.Replace(template)
}

func GetDockerfileFromConfig(path string) string {
	return ParseDockerfileTemplate(path)
}
