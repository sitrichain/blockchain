package java

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/klauspost/compress/gzip"
	"github.com/rongzer/blockchain/common/conf"
	cutil "github.com/rongzer/blockchain/peer/container/util"
	pb "github.com/rongzer/blockchain/protos/peer"
)

// Platform for java chaincodes in java
type Platform struct {
}

var buildCmds = map[string]string{
	"src/build.gradle": "gradle -b build.gradle clean && gradle -b build.gradle build",
	"src/pom.xml":      "mvn -f pom.xml clean && mvn -f pom.xml package",
}

//getBuildCmd returns the type of build gradle/maven based on the file
//found in java chaincode project root
//build.gradle - gradle  - returns the first found build type
//pom.xml - maven
func getBuildCmd(codePackage []byte) (string, error) {

	is := bytes.NewReader(codePackage)
	gr, err := gzip.NewReader(is)
	if err != nil {
		return "", fmt.Errorf("failure opening gzip stream: %s", err)
	}
	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err != nil {
			return "", errors.New("Build file not found")
		}

		if cmd, ok := buildCmds[header.Name]; ok == true {
			return cmd, nil
		}
	}
}

//ValidateSpec validates the java chaincode specs
func (javaPlatform *Platform) ValidateSpec(spec *pb.ChaincodeSpec) error {
	path, err := url.Parse(spec.ChaincodeId.Path)
	if err != nil || path == nil {
		return fmt.Errorf("invalid path: %s", err)
	}

	//we have no real good way of checking existence of remote urls except by downloading and testing
	//which we do later anyway. But we *can* - and *should* - test for existence of local paths.
	//Treat empty scheme as a local filesystem path
	//	if url.Scheme == "" {
	//		pathToCheck := filepath.Join(os.Getenv("GOPATH"), "src", spec.ChaincodeId.Path)
	//		exists, err := pathExists(pathToCheck)
	//		if err != nil {
	//			return fmt.Errorf("Error validating chaincode path: %s", err)
	//		}
	//		if !exists {
	//			return fmt.Errorf("Path to chaincode does not exist: %s", spec.ChaincodeId.Path)
	//		}
	//	}
	return nil
}

func (javaPlatform *Platform) ValidateDeploymentSpec(_ *pb.ChaincodeDeploymentSpec) error {
	// FIXME: Java platform needs to implement its own validation similar to GOLANG
	return nil
}

// WritePackage writes the java chaincode package
func (javaPlatform *Platform) GetDeploymentPayload(spec *pb.ChaincodeSpec) ([]byte, error) {

	var err error

	inputbuf := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(inputbuf)
	tw := tar.NewWriter(gw)

	//ignore the generated hash. Just use the tw
	//The hash could be used in a future enhancement
	//to check, warn of duplicate installs etc.
	_, err = collectChaincodeFiles(spec, tw)
	if err != nil {
		return nil, err
	}

	err = writeChaincodePackage(spec, tw)

	tw.Close()
	gw.Close()

	if err != nil {
		return nil, err
	}

	payload := inputbuf.Bytes()

	return payload, nil
}

func (javaPlatform *Platform) GenerateDockerfile(cds *pb.ChaincodeDeploymentSpec) (string, error) {
	var err error
	var buf []string

	buildCmd, err := getBuildCmd(cds.CodePackage)
	if err != nil {
		return "", err
	}

	buf = append(buf, conf.V.Peer.Chaincode.JavaDockerfile)
	buf = append(buf, "ADD codepackage.tgz /root/chaincode")
	buf = append(buf, "RUN  cd /root/chaincode/src && "+buildCmd)
	buf = append(buf, "RUN  cp /root/chaincode/src/build/chaincode.jar /root")
	buf = append(buf, "RUN  cp /root/chaincode/src/build/libs/* /root/libs")

	dockerFileContents := strings.Join(buf, "\n")

	return dockerFileContents, nil
}

func (javaPlatform *Platform) GenerateDockerBuild(cds *pb.ChaincodeDeploymentSpec, tw *tar.Writer) error {
	return cutil.WriteBytesToPackage("codepackage.tgz", cds.CodePackage, tw)
}
