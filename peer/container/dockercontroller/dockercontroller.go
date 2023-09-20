package dockercontroller

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/log"
	container "github.com/rongzer/blockchain/peer/container/api"
	"github.com/rongzer/blockchain/peer/container/ccintf"
	cutil "github.com/rongzer/blockchain/peer/container/util"
	pb "github.com/rongzer/blockchain/protos/peer"
	"golang.org/x/net/context"
)

// getClient returns an instance that implements dockerClient interface
type getClient func() (dockerClient, error)

//DockerVM is a vm. It is identified by an image id
type DockerVM struct {
	id           string
	getClientFnc getClient
}

// dockerClient represents a docker client
type dockerClient interface {
	// CreateContainer creates a docker container, returns an error in case of failure
	CreateContainer(opts docker.CreateContainerOptions) (*docker.Container, error)
	// StartContainer starts a docker container, returns an error in case of failure
	StartContainer(id string, cfg *docker.HostConfig) error
	// AttachToContainer attaches to a docker container, returns an error in case of
	// failure
	AttachToContainer(opts docker.AttachToContainerOptions) error
	// BuildImage builds an image from a tarball's url or a Dockerfile in the input
	// stream, returns an error in case of failure
	BuildImage(opts docker.BuildImageOptions) error
	// RemoveImageExtended removes a docker image by its name or ID, returns an
	// error in case of failure
	RemoveImageExtended(id string, opts docker.RemoveImageOptions) error
	// StopContainer stops a docker container, killing it after the given timeout
	// (in seconds). Returns an error in case of failure
	StopContainer(id string, timeout uint) error
	// KillContainer sends a signal to a docker container, returns an error in
	// case of failure
	KillContainer(opts docker.KillContainerOptions) error
	// RemoveContainer removes a docker container, returns an error in case of failure
	RemoveContainer(opts docker.RemoveContainerOptions) error
	// 导入Images
	LoadImage(opts docker.LoadImageOptions) error
}

// NewDockerVM returns a new DockerVM instance
func NewDockerVM() *DockerVM {
	vm := DockerVM{}
	vm.getClientFnc = getDockerClient
	return &vm
}

func getDockerClient() (dockerClient, error) {
	return cutil.NewDockerClient()
}

func getDockerHostConfig() *docker.HostConfig {
	log.Logger.Debugf("docker container hostconfig NetworkMode: %s", conf.V.Peer.VM.HostConfig.NetworkMode)

	conf.V.Peer.VM.HostConfig.MemorySwappiness = &conf.V.Peer.VM.MemorySwappiness
	conf.V.Peer.VM.HostConfig.OOMKillDisable = &conf.V.Peer.VM.OOMKillDisable

	return conf.V.Peer.VM.HostConfig
}

func (vm *DockerVM) createContainer(_ context.Context, client dockerClient,
	imageID string, containerID string, args []string,
	env []string, attachStdout bool) error {
	config := docker.Config{Cmd: args, Image: imageID, Env: env, AttachStdout: attachStdout, AttachStderr: attachStdout}
	copts := docker.CreateContainerOptions{Name: containerID, Config: &config, HostConfig: getDockerHostConfig()}
	log.Logger.Debugf("Create container: %s", imageID)
	_, err := client.CreateContainer(copts)
	if err != nil {
		return err
	}
	log.Logger.Debugf("Created container: %s", imageID)
	return nil
}

func (vm *DockerVM) deployImage(client dockerClient, ccid ccintf.CCID,
	_ []string, _ []string, reader io.Reader) error {
	id, err := vm.GetVMName(ccid)
	if err != nil {
		return err
	}
	outputbuf := bytes.NewBuffer(nil)
	opts := docker.BuildImageOptions{
		Name:         id,
		Pull:         false,
		InputStream:  reader,
		OutputStream: outputbuf,
	}

	if err := client.BuildImage(opts); err != nil {
		log.Logger.Errorf("Error building images: %s", err)
		log.Logger.Errorf("Image Output:\n********************\n%s\n********************", outputbuf.String())
		return err
	}

	log.Logger.Debugf("Created image: %s", id)

	return nil
}

//Deploy use the reader containing targz to create a docker image
//for docker inputbuf is tar reader ready for use by docker.Client
//the stream from end client to peer could directly be this tar stream
//talk to docker daemon using docker Client and build the image
func (vm *DockerVM) Deploy(_ context.Context, ccid ccintf.CCID,
	args []string, env []string, reader io.Reader) error {

	client, err := vm.getClientFnc()
	switch err {
	case nil:
		if err = vm.deployImage(client, ccid, args, env, reader); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Error creating docker client: %s", err)
	}
	return nil
}

//Start starts a container using a previously created docker image
func (vm *DockerVM) Start(ctxt context.Context, ccid ccintf.CCID,
	args []string, env []string, builder container.BuildSpecFactory, prelaunchFunc container.PrelaunchFunc) error {
	imageID, err := vm.GetVMName(ccid)
	if err != nil {
		return err
	}

	//log.Logger.Infof("imageId : %s", imageID)

	client, err := vm.getClientFnc()
	if err != nil {
		log.Logger.Debugf("start - cannot create client %s", err)
		return err
	}

	containerID := strings.Replace(imageID, ":", "_", -1)

	if len(args) > 7 {
		chainId := args[7]
		if ccid.NetworkID != "" {
			containerID = strings.Replace(containerID, ccid.NetworkID, chainId, 1)
		}

		log.Logger.Infof("chainId : %s containnerId : %s", chainId, containerID)
	}

	attachStdout := conf.V.Peer.VM.AttachStdout

	//log.Logger.Infof("containerID : %s", containerID)

	//stop,force remove if necessary
	log.Logger.Debugf("Cleanup container %s", containerID)
	vm.stopInternal(ctxt, client, containerID, 0, false, false)

	log.Logger.Debugf("Start container %s", containerID)
	log.Logger.Infof("execute args1:%s, %s,%s, spec:%v,%v,%v,%s", imageID, containerID, strings.Join(args, ","), ccid.ChaincodeSpec, ccid.ChaincodeSpec.Type, pb.ChaincodeSpec_JAVA, ccid.ChaincodeSpec.Type == pb.ChaincodeSpec_JAVA)

	if ccid.ChaincodeSpec.Type == pb.ChaincodeSpec_JAVA && len(args) > 5 && args[4] == "chaincode.jar" {
		javaEnv := conf.V.Peer.Chaincode.JavaDockerfile

		javaEnv = strings.Replace(javaEnv, "from ", "", -1)
		javaEnv = strings.Replace(javaEnv, " ", "", -1)
		javaEnv = strings.Replace(javaEnv, "\n", "", -1)
		dockerHubDomain := conf.V.Peer.VM.Hub
		if len(dockerHubDomain) < 10 {
			dockerHubDomain = ""
		} else {
			dockerHubDomain = strings.TrimSpace(dockerHubDomain)
		}
		javaEnv = dockerHubDomain + javaEnv

		imageID = javaEnv

		args[4] = "libs/shim-client-1.0.jar"
		log.Logger.Infof("execute args4:%s", javaEnv)
		//imageID = javaEnv
	}

	err = vm.createContainer(ctxt, client, imageID, containerID, args, env, attachStdout)
	if err != nil {
		//if image not found try to create image and retry
		if err == docker.ErrNoSuchImage {
			if builder != nil {
				log.Logger.Debugf("start-could not find image <%s> (container id <%s>), because of <%s>..."+
					"attempt to recreate image", imageID, containerID, err)

				reader, err1 := builder()
				if err1 != nil {
					log.Logger.Errorf("Error creating image builder for image <%s> (container id <%s>), "+
						"because of <%s>", imageID, containerID, err1)
				}

				if err1 = vm.deployImage(client, ccid, args, env, reader); err1 != nil {
					return err1
				}

				log.Logger.Debug("start-recreated image successfully")
				if err1 = vm.createContainer(ctxt, client, imageID, containerID, args, env, attachStdout); err1 != nil {
					log.Logger.Errorf("start-could not recreate container post recreate image: %s", err1)
					return err1
				}
			} else {
				log.Logger.Errorf("start-could not find image <%s>, because of %s", imageID, err)
				return err
			}
		} else {
			log.Logger.Errorf("start-could not recreate container <%s>, because of %s", containerID, err)
			return err
		}
	}

	if attachStdout {
		// Launch a few go-threads to manage output streams from the container.
		// They will be automatically destroyed when the container exits
		attached := make(chan struct{})
		r, w := io.Pipe()

		go func() {
			// AttachToContainer will fire off a message on the "attached" channel once the
			// attachment completes, and then block until the container is terminated.
			// The returned error is not used outside the scope of this function. Assign the
			// error to a local variable to prevent clobbering the function variable 'err'.
			err := client.AttachToContainer(docker.AttachToContainerOptions{
				Container:    containerID,
				OutputStream: w,
				ErrorStream:  w,
				Logs:         true,
				Stdout:       true,
				Stderr:       true,
				Stream:       true,
				Success:      attached,
			})

			// If we get here, the container has terminated.  Send a signal on the pipe
			// so that downstream may clean up appropriately
			_ = w.CloseWithError(err)
		}()

		go func() {
			// Block here until the attachment completes or we timeout
			select {
			case <-attached:
				// successful attach
			case <-time.After(10 * time.Second):
				log.Logger.Errorf("Timeout while attaching to IO channel in container %s", containerID)
				return
			}

			// Acknowledge the attachment?  This was included in the gist I followed
			// (http://bit.ly/2jBrCtM).  Not sure it's actually needed but it doesn't
			// appear to hurt anything.
			attached <- struct{}{}

			// Establish a buffer for our IO channel so that we may do readline-style
			// ingestion of the IO, one log entry per line
			is := bufio.NewReader(r)

			for {
				// Loop forever dumping lines of text into the log.Logger
				// until the pipe is closed
				line, err2 := is.ReadString('\n')
				if err2 != nil {
					switch err2 {
					case io.EOF:
						log.Logger.Infof("Container %s has closed its IO channel", containerID)
					default:
						log.Logger.Errorf("Error reading container output: %s", err2)
					}

					return
				}

				log.Logger.Info(line)
			}
		}()
	}

	if prelaunchFunc != nil {
		if err = prelaunchFunc(); err != nil {
			return err
		}
	}

	// start container with HostConfig was deprecated since v1.10 and removed in v1.2
	err = client.StartContainer(containerID, nil)
	if err != nil {
		log.Logger.Errorf("start-could not start container: %s", err)
		return err
	}

	log.Logger.Debugf("Started container %s", containerID)
	return nil
}

//Stop stops a running chaincode
func (vm *DockerVM) Stop(ctxt context.Context, ccid ccintf.CCID, timeout uint, dontkill bool, dontremove bool) error {
	id, err := vm.GetVMName(ccid)
	if err != nil {
		return err
	}

	client, err := vm.getClientFnc()
	if err != nil {
		log.Logger.Debugf("stop - cannot create client %s", err)
		return err
	}
	id = strings.Replace(id, ":", "_", -1)

	err = vm.stopInternal(ctxt, client, id, timeout, dontkill, dontremove)

	return err
}

func (vm *DockerVM) stopInternal(_ context.Context, client dockerClient,
	id string, timeout uint, dontkill bool, dontremove bool) error {
	err := client.StopContainer(id, timeout)
	if err != nil {
		log.Logger.Debugf("Stop container %s(%s)", id, err)
	} else {
		log.Logger.Debugf("Stopped container %s", id)
	}
	if !dontkill {
		err = client.KillContainer(docker.KillContainerOptions{ID: id})
		if err != nil {
			log.Logger.Debugf("Kill container %s (%s)", id, err)
		} else {
			log.Logger.Debugf("Killed container %s", id)
		}
	}
	if !dontremove {
		err = client.RemoveContainer(docker.RemoveContainerOptions{ID: id, Force: true})
		if err != nil {
			log.Logger.Debugf("Remove container %s (%s)", id, err)
		} else {
			log.Logger.Debugf("Removed container %s", id)
		}
	}
	return err
}

//Destroy destroys an image
func (vm *DockerVM) Destroy(_ context.Context, ccid ccintf.CCID, force bool, noprune bool) error {
	id, err := vm.GetVMName(ccid)
	if err != nil {
		return err
	}

	client, err := vm.getClientFnc()
	if err != nil {
		log.Logger.Errorf("destroy-cannot create client %s", err)
		return err
	}
	id = strings.Replace(id, ":", "_", -1)

	err = client.RemoveImageExtended(id, docker.RemoveImageOptions{Force: force, NoPrune: noprune})

	if err != nil {
		log.Logger.Errorf("error while destroying image: %s", err)
	} else {
		log.Logger.Debugf("Destroyed image %s", id)
	}

	return err
}

//GetVMName generates the docker image from peer information given the hashcode. This is needed to
//keep image name's unique in a single host, multi-peer environment (such as a development environment)
func (vm *DockerVM) GetVMName(ccid ccintf.CCID) (string, error) {
	name := ccid.GetName()

	//log.Logger.Infof("ccid.GetName() : %s networkid : %s peerid : %s", name, ccid.NetworkID, ccid.PeerID)

	//Convert to lowercase and replace any invalid characters with "-"
	r := regexp.MustCompile("[^a-zA-Z0-9-_.]")

	if ccid.NetworkID != "" && ccid.PeerID != "" {
		name = strings.ToLower(
			r.ReplaceAllString(
				fmt.Sprintf("%s-%s-%s", ccid.NetworkID, ccid.PeerID, name), "-"))
	} else if ccid.NetworkID != "" {
		name = strings.ToLower(
			r.ReplaceAllString(
				fmt.Sprintf("%s-%s", ccid.NetworkID, name), "-"))
	} else if ccid.PeerID != "" {
		name = strings.ToLower(
			r.ReplaceAllString(
				fmt.Sprintf("%s-%s", ccid.PeerID, name), "-"))
	}

	// Check name complies with Docker's repository naming rules
	r = regexp.MustCompile("^[a-z0-9]+(([._-][a-z0-9]+)+)?$")

	if !r.MatchString(name) {
		log.Logger.Errorf("Error constructing Docker VM Name. '%s' breaks Docker's repository naming rules", name)
		return name, fmt.Errorf("Error constructing Docker VM Name. '%s' breaks Docker's repository naming rules", name)
	}
	return name, nil
}
