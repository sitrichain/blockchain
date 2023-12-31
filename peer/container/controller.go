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

package container

import (
	"fmt"
	"io"
	"sync"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/container/api"
	"github.com/rongzer/blockchain/peer/container/ccintf"
	"github.com/rongzer/blockchain/peer/container/dockercontroller"
	"github.com/rongzer/blockchain/peer/container/inproccontroller"
	"golang.org/x/net/context"
)

type refCountedLock struct {
	refCount int
	lock     *sync.RWMutex
}

//VMController - manages VMs
//   . abstract construction of different types of VMs (we only care about Docker for now)
//   . manage lifecycle of VM (start with build, start, stop ...
//     eventually probably need fine grained management)
type VMController struct {
	sync.RWMutex
	// Handlers for each chaincode
	containerLocks map[string]*refCountedLock
}

//singleton...acess through NewVMController
var vmcontroller *VMController

//constants for supported containers
const (
	DOCKER = "Docker"
	SYSTEM = "System"
)

//NewVMController - creates/returns singleton
func init() {
	vmcontroller = new(VMController)
	vmcontroller.containerLocks = make(map[string]*refCountedLock)
}

func (vmc *VMController) newVM(typ string) api.VM {
	var (
		v api.VM
	)

	switch typ {
	case DOCKER:
		v = dockercontroller.NewDockerVM()
	case SYSTEM:
		v = &inproccontroller.InprocVM{}
	default:
		v = &dockercontroller.DockerVM{}
	}
	return v
}

func (vmc *VMController) lockContainer(id string) {
	//get the container lock under global lock
	vmcontroller.Lock()
	var refLck *refCountedLock
	var ok bool
	if refLck, ok = vmcontroller.containerLocks[id]; !ok {
		refLck = &refCountedLock{refCount: 1, lock: &sync.RWMutex{}}
		vmcontroller.containerLocks[id] = refLck
	} else {
		refLck.refCount++
		log.Logger.Debugf("refcount %d (%s)", refLck.refCount, id)
	}
	vmcontroller.Unlock()
	log.Logger.Debugf("waiting for container(%s) lock", id)
	refLck.lock.Lock()
	log.Logger.Debugf("got container (%s) lock", id)
}

func (vmc *VMController) unlockContainer(id string) {
	vmcontroller.Lock()
	if refLck, ok := vmcontroller.containerLocks[id]; ok {
		if refLck.refCount <= 0 {
			panic("refcnt <= 0")
		}
		refLck.lock.Unlock()
		if refLck.refCount--; refLck.refCount == 0 {
			log.Logger.Debugf("container lock deleted(%s)", id)
			delete(vmcontroller.containerLocks, id)
		}
	} else {
		log.Logger.Debugf("no lock to unlock(%s)!!", id)
	}
	vmcontroller.Unlock()
}

//VMCReqIntf - all requests should implement this interface.
//The context should be passed and tested at each layer till we stop
//note that we'd stop on the first method on the stack that does not
//take context
type VMCReqIntf interface {
	do(ctxt context.Context, v api.VM) VMCResp
	getCCID() ccintf.CCID
}

//VMCResp - response from requests. resp field is a anon interface.
//It can hold any response. err should be tested first
type VMCResp struct {
	Err  error
	Resp interface{}
}

//CreateImageReq - properties for creating an container image
type CreateImageReq struct {
	ccintf.CCID
	Reader io.Reader
	Args   []string
	Env    []string
}

func (bp CreateImageReq) do(ctxt context.Context, v api.VM) VMCResp {
	var resp VMCResp

	if err := v.Deploy(ctxt, bp.CCID, bp.Args, bp.Env, bp.Reader); err != nil {
		resp = VMCResp{Err: err}
	} else {
		resp = VMCResp{}
	}

	return resp
}

func (bp CreateImageReq) getCCID() ccintf.CCID {
	return bp.CCID
}

//StartImageReq - properties for starting a container.
type StartImageReq struct {
	ccintf.CCID
	Builder       api.BuildSpecFactory
	Args          []string
	Env           []string
	PrelaunchFunc api.PrelaunchFunc
}

func (si StartImageReq) do(ctxt context.Context, v api.VM) VMCResp {
	var resp VMCResp

	if err := v.Start(ctxt, si.CCID, si.Args, si.Env, si.Builder, si.PrelaunchFunc); err != nil {
		resp = VMCResp{Err: err}
	} else {
		resp = VMCResp{}
	}

	return resp
}

func (si StartImageReq) getCCID() ccintf.CCID {
	return si.CCID
}

//StopImageReq - properties for stopping a container.
type StopImageReq struct {
	ccintf.CCID
	Timeout uint
	//by default we will kill the container after stopping
	Dontkill bool
	//by default we will remove the container after killing
	Dontremove bool
}

func (si StopImageReq) do(ctxt context.Context, v api.VM) VMCResp {
	var resp VMCResp

	if err := v.Stop(ctxt, si.CCID, si.Timeout, si.Dontkill, si.Dontremove); err != nil {
		resp = VMCResp{Err: err}
	} else {
		resp = VMCResp{}
	}

	return resp
}

func (si StopImageReq) getCCID() ccintf.CCID {
	return si.CCID
}

//DestroyImageReq - properties for stopping a container.
type DestroyImageReq struct {
	ccintf.CCID
	Timeout uint
	Force   bool
	NoPrune bool
}

func (di DestroyImageReq) do(ctxt context.Context, v api.VM) VMCResp {
	var resp VMCResp

	if err := v.Destroy(ctxt, di.CCID, di.Force, di.NoPrune); err != nil {
		resp = VMCResp{Err: err}
	} else {
		resp = VMCResp{}
	}

	return resp
}

func (di DestroyImageReq) getCCID() ccintf.CCID {
	return di.CCID
}

//VMCProcess should be used as follows
//   . construct a context
//   . construct req of the right type (e.g., CreateImageReq)
//   . call it in a go routine
//   . process response in the go routing
//context can be cancelled. VMCProcess will try to cancel calling functions if it can
//For instance docker clients api's such as BuildImage are not cancelable.
//In all cases VMCProcess will wait for the called go routine to return
func VMCProcess(ctxt context.Context, vmtype string, req VMCReqIntf) (interface{}, error) {
	// vmtype inproc docker
	v := vmcontroller.newVM(vmtype)

	if v == nil {
		return nil, fmt.Errorf("Unknown VM type %s", vmtype)
	}

	c := make(chan struct{})
	var resp interface{}
	go func() {
		defer close(c)

		id, err := v.GetVMName(req.getCCID())
		if err != nil {
			resp = VMCResp{Err: err}
			return
		}
		vmcontroller.lockContainer(id)
		resp = req.do(ctxt, v)
		vmcontroller.unlockContainer(id)
	}()

	select {
	case <-c:
		return resp, nil
	case <-ctxt.Done():
		//TODO cancel req.do ... (needed) ?
		<-c
		return nil, ctxt.Err()
	}
}
