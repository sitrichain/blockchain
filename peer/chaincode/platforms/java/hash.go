/*
Copyright DTCC 2016 All Rights Reserved.

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

package java

import (
	"archive/tar"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/util"
	ccutil "github.com/rongzer/blockchain/peer/chaincode/platforms/util"
	pb "github.com/rongzer/blockchain/protos/peer"
)

//collectChaincodeFiles collects chaincode files and generates hashcode for the
//package.
//NOTE: for dev mode, user builds and runs chaincode manually. The name provided
//by the user is equivalent to the path. This method will treat the name
//as codebytes and compute the hash from it. ie, user cannot run the chaincode
//with the same (name, input, args)
func collectChaincodeFiles(spec *pb.ChaincodeSpec, tw *tar.Writer) (string, error) {
	if spec == nil {
		return "", errors.New("Cannot collect chaincode files from nil spec")
	}

	chaincodeID := spec.ChaincodeId
	if chaincodeID == nil || chaincodeID.Path == "" {
		return "", errors.New("Cannot collect chaincode files from empty chaincode path")
	}

	codepath := chaincodeID.Path

	var err error
	if !strings.HasPrefix(codepath, "/") {
		wd := ""
		wd, err = os.Getwd()
		codepath = wd + "/" + codepath
	}

	if err != nil {
		return "", fmt.Errorf("Error getting code %s", err)
	}

	if err = ccutil.IsCodeExist(codepath); err != nil {
		return "", fmt.Errorf("code does not exist %s", err)
	}

	var hash []byte

	//install will not have inputs and we don't have to collect hash for it
	if spec.Input == nil || len(spec.Input.Args) == 0 {
		log.Logger.Debugf("not using input for hash computation for %v ", chaincodeID)
	} else {
		inputbytes, err2 := spec.Input.Marshal()
		if err2 != nil {
			return "", fmt.Errorf("Error marshalling constructor: %s", err)
		}
		hash = util.GenerateHashFromSignature(inputbytes)
	}

	hash, err = ccutil.HashFilesInDir("", codepath, hash, tw)
	if err != nil {
		return "", fmt.Errorf("could not get hashcode for %s - %s", codepath, err)
	}

	return hex.EncodeToString(hash[:]), nil
}
