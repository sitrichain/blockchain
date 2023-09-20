package car

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/peer/chaincode/platforms/util"
	cutil "github.com/rongzer/blockchain/peer/container/util"
	pb "github.com/rongzer/blockchain/protos/peer"
)

// Platform for the CAR type
type Platform struct {
}

// ValidateSpec validates the chaincode specification for CAR types to satisfy
// the platform interface.  This chaincode type currently doesn't
// require anything specific so we just implicitly approve any spec
func (carPlatform *Platform) ValidateSpec(_ *pb.ChaincodeSpec) error {
	return nil
}

func (carPlatform *Platform) ValidateDeploymentSpec(_ *pb.ChaincodeDeploymentSpec) error {
	// CAR platform will validate the code package within chaintool
	return nil
}

func (carPlatform *Platform) GetDeploymentPayload(spec *pb.ChaincodeSpec) ([]byte, error) {

	return ioutil.ReadFile(spec.ChaincodeId.Path)
}

func (carPlatform *Platform) GenerateDockerfile(_ *pb.ChaincodeDeploymentSpec) (string, error) {

	var buf []string

	//let the executable's name be chaincode ID's name
	buf = append(buf, "FROM "+conf.V.Peer.Chaincode.CarRuntime)
	buf = append(buf, "ADD binpackage.tar /usr/local/bin")

	dockerFileContents := strings.Join(buf, "\n")

	return dockerFileContents, nil
}

func (carPlatform *Platform) GenerateDockerBuild(cds *pb.ChaincodeDeploymentSpec, tw *tar.Writer) error {

	// Bundle the .car file into a tar stream so it may be transferred to the builder container
	codepackage, output := io.Pipe()
	go func() {
		tw := tar.NewWriter(output)

		err := cutil.WriteBytesToPackage("codepackage.car", cds.CodePackage, tw)

		tw.Close()
		output.CloseWithError(err)
	}()

	binpackage := bytes.NewBuffer(nil)
	err := util.DockerBuild(util.DockerBuildOptions{
		Cmd:          "java -jar /usr/local/bin/chaintool buildcar /chaincode/input/codepackage.car -o /chaincode/output/chaincode",
		InputStream:  codepackage,
		OutputStream: binpackage,
	})
	if err != nil {
		return fmt.Errorf("Error building CAR: %s", err)
	}

	return cutil.WriteBytesToPackage("binpackage.tar", binpackage.Bytes(), tw)
}
