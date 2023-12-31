/*
Copyright IBM Corp. 2016 All Rights Reserved.

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

package inproccontroller

import (
	"fmt"
	"io"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	container "github.com/rongzer/blockchain/peer/container/api"
	"github.com/rongzer/blockchain/peer/container/ccintf"
	pb "github.com/rongzer/blockchain/protos/peer"
	"golang.org/x/net/context"
)

type inprocContainer struct {
	chaincode shim.Chaincode
	running   bool
	args      []string
	env       []string
	stopChan  chan struct{}
}

var (
	typeRegistry = make(map[string]*inprocContainer)
	instRegistry = make(map[string]*inprocContainer)
)

// errors

//SysCCRegisteredErr registered error
type SysCCRegisteredErr string

func (s SysCCRegisteredErr) Error() string {
	return fmt.Sprintf("%s already registered", string(s))
}

//Register registers system chaincode with given path. The deploy should be called to initialize
func Register(path string, cc shim.Chaincode) error {
	tmp := typeRegistry[path]
	if tmp != nil {
		return SysCCRegisteredErr(path)
	}

	typeRegistry[path] = &inprocContainer{chaincode: cc}
	return nil
}

//InprocVM is a vm. It is identified by a executable name
type InprocVM struct {
	id string
}

func (vm *InprocVM) getInstance(_ context.Context, ipctemplate *inprocContainer, instName string, args []string, env []string) (*inprocContainer, error) {
	ipc := instRegistry[instName]
	if ipc != nil {
		log.Logger.Warnf("chaincode instance exists for %s", instName)
		return ipc, nil
	}
	ipc = &inprocContainer{args: args, env: env, chaincode: ipctemplate.chaincode, stopChan: make(chan struct{})}
	instRegistry[instName] = ipc
	log.Logger.Debugf("chaincode instance created for %s", instName)
	return ipc, nil
}

//Deploy verifies chaincode is registered and creates an instance for it. Currently only one instance can be created
func (vm *InprocVM) Deploy(ctxt context.Context, ccid ccintf.CCID, args []string, env []string, _ io.Reader) error {
	path := ccid.ChaincodeSpec.ChaincodeId.Path

	ipctemplate := typeRegistry[path]
	if ipctemplate == nil {
		return fmt.Errorf(fmt.Sprintf("%s not registered. Please register the system chaincode in inprocinstances.go", path))
	}

	if ipctemplate.chaincode == nil {
		return fmt.Errorf(fmt.Sprintf("%s system chaincode does not contain chaincode instance", path))
	}

	instName, _ := vm.GetVMName(ccid)
	_, err := vm.getInstance(ctxt, ipctemplate, instName, args, env)

	//FUTURE ... here is where we might check code for safety
	log.Logger.Debugf("registered : %s", path)

	return err
}

func (ipc *inprocContainer) launchInProc(ctxt context.Context, id string, args []string, env []string, ccSupport ccintf.CCSupport) error {
	peerRcvCCSend := make(chan *pb.ChaincodeMessage)
	ccRcvPeerSend := make(chan *pb.ChaincodeMessage)
	var err error
	ccchan := make(chan struct{}, 1)
	ccsupportchan := make(chan struct{}, 1)
	go func() {
		defer close(ccchan)
		log.Logger.Infof("chaincode started for %s", id)
		if args == nil {
			args = ipc.args
		}
		if env == nil {
			env = ipc.env
		}
		// 系统合约启动，获取读写流
		err := shim.StartInProc(env, args, ipc.chaincode, ccRcvPeerSend, peerRcvCCSend)
		if err != nil {
			err = fmt.Errorf("chaincode-support ended with err: %s", err)
			log.Logger.Errorf("%s", err)
		}
		log.Logger.Debugf("chaincode ended with for  %s with err: %s", id, err)
	}()

	go func() {
		defer close(ccsupportchan)
		inprocStream := newInProcStream(peerRcvCCSend, ccRcvPeerSend)
		log.Logger.Infof("chaincode-support started for  %s", id)
		err := ccSupport.HandleChaincodeStream(ctxt, inprocStream)
		if err != nil {
			err = fmt.Errorf("chaincode ended with err: %s", err)
			log.Logger.Errorf("%s", err)
		}
		log.Logger.Debugf("chaincode-support ended with for  %s with err: %s", id, err)
	}()

	select {
	case <-ccchan:
		close(peerRcvCCSend)
		log.Logger.Warnf("chaincode %s quit", id)
	case <-ccsupportchan:
		close(ccRcvPeerSend)
		log.Logger.Warnf("chaincode support %s quit", id)
	case <-ipc.stopChan:
		close(ccRcvPeerSend)
		close(peerRcvCCSend)
		log.Logger.Warnf("chaincode %s stopped", id)
	}

	return err
}

//Start starts a previously registered system codechain
func (vm *InprocVM) Start(ctxt context.Context, ccid ccintf.CCID, args []string, env []string, _ container.BuildSpecFactory, prelaunchFunc container.PrelaunchFunc) error {
	path := ccid.ChaincodeSpec.ChaincodeId.Path

	ipctemplate := typeRegistry[path]

	if ipctemplate == nil {
		return fmt.Errorf(fmt.Sprintf("%s not registered", path))
	}

	instName, _ := vm.GetVMName(ccid)

	ipc, err := vm.getInstance(ctxt, ipctemplate, instName, args, env)

	if err != nil {
		return fmt.Errorf(fmt.Sprintf("could not create instance for %s", instName))
	}

	if ipc.running {
		return fmt.Errorf(fmt.Sprintf("chaincode running %s", path))
	}

	//TODO VALIDITY CHECKS ?

	ccSupport, ok := ctxt.Value(ccintf.GetCCHandlerKey()).(ccintf.CCSupport)
	if !ok || ccSupport == nil {
		return fmt.Errorf("in-process communication generator not supplied")
	}

	if prelaunchFunc != nil {
		if err = prelaunchFunc(); err != nil {
			return err
		}
	}

	ipc.running = true

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Logger.Errorf("caught panic from chaincode  %s", instName)
			}
		}()
		ipc.launchInProc(ctxt, instName, args, env, ccSupport)
	}()

	return nil
}

//Stop stops a system codechain
func (vm *InprocVM) Stop(_ context.Context, ccid ccintf.CCID, _ uint, _ bool, _ bool) error {
	path := ccid.ChaincodeSpec.ChaincodeId.Path

	ipctemplate := typeRegistry[path]
	if ipctemplate == nil {
		return fmt.Errorf("%s not registered", path)
	}

	instName, _ := vm.GetVMName(ccid)

	ipc := instRegistry[instName]

	if ipc == nil {
		return fmt.Errorf("%s not found", instName)
	}

	if !ipc.running {
		return fmt.Errorf("%s not running", instName)
	}

	ipc.stopChan <- struct{}{}

	delete(instRegistry, instName)
	//TODO stop
	return nil
}

//Destroy destroys an image
func (vm *InprocVM) Destroy(_ context.Context, _ ccintf.CCID, _ bool, _ bool) error {
	//not implemented
	return nil
}

//GetVMName ignores the peer and network name as it just needs to be unique in process
func (vm *InprocVM) GetVMName(ccid ccintf.CCID) (string, error) {
	return ccid.GetName(), nil
}
