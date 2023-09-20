# This makefile defines the following targets
#
#   - all (default) - builds all targets and runs all tests/checks
#   - configtxgen - builds a native configtxgen binary
#   - cryptogen  -  builds a native cryptogen binary
#   - peer - builds a native fabric peer binary
#   - orderer - builds a native fabric orderer binary
#   - clean - cleans the build area

PROJECT_NAME = rongzer/blockchain
BASE_VERSION = 1.0.0
PREV_VERSION = 1.0.0-rc1
GIT_VERSION=$(shell git symbolic-ref --short HEAD)
BUILD_TIME=$(shell date "+%Y/%m/%d_%H:%M:%S")
IS_RELEASE = true

ifneq ($(IS_RELEASE),true)
EXTRA_VERSION ?= snapshot-$(shell git rev-parse --short HEAD)
PROJECT_VERSION=$(BASE_VERSION)-$(EXTRA_VERSION)
else
PROJECT_VERSION=$(BASE_VERSION)
endif

PKGNAME = github.com/$(PROJECT_NAME)
CGO_FLAGS = CGO_CFLAGS=" "
ARCH=$(shell uname -m)
MARCH=$(shell go env GOOS)-$(shell go env GOARCH)
CHAINTOOL_RELEASE=v0.10.3
BASEIMAGE_RELEASE=0.3.1

SCALA_VERSION=2.12
KAFKA_VERSION=2.3.0

BASE_DOCKER_LABEL=org.rongzer.blockchain

# defined in common/metadata/metadata.go
METADATA_VAR = Version=$(PROJECT_VERSION)
METADATA_VAR += BaseVersion=$(BASEIMAGE_RELEASE)
METADATA_VAR += BaseDockerLabel=$(BASE_DOCKER_LABEL)
METADATA_VAR += DockerNamespace=$(DOCKER_NS)
METADATA_VAR += BaseDockerNamespace=$(BASE_DOCKER_NS)
METADATA_VAR += GitVersion=$(GIT_VERSION)
METADATA_VAR += BuildTime=$(BUILD_TIME)

GO_LDFLAGS = $(patsubst %,-X $(PKGNAME)/common/metadata.%,$(METADATA_VAR))

GO_TAGS ?=

CHAINTOOL_URL ?= https://github.com/hyperledger/fabric-chaintool/releases/download/$(CHAINTOOL_RELEASE)/chaintool

export GO_LDFLAGS

EXECUTABLES = git
K := $(foreach exec,$(EXECUTABLES),\
	$(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH: Check dependencies")))

GOSHIM_DEPS = $(shell ./scripts/goListFiles.sh $(PKGNAME)/core/chaincode/shim)
JAVASHIM_DEPS =  $(shell git ls-files peer/chaincode/shim/java -c -o)
PROTOS = $(shell git ls-files *.proto | grep -v vendor)
# No sense rebuilding when non production code is changed
PROJECT_FILES = $(shell git ls-files  | grep -v ^test | grep -v ^unit-test | \
	grep -v ^bddtests | grep -v ^docs | grep -v _test.go$ | grep -v .md$ | \
	grep -v ^.git | grep -v ^examples | grep -v ^devenv | grep -v .png$ | \
	grep -v ^LICENSE )
RELEASE_TEMPLATES = $(shell git ls-files | grep "release/templates")

IMAGES = peer light orderer javaenv

RELEASE_PLATFORMS = windows-amd64 darwin-amd64 linux-amd64 linux-ppc64le linux-s390x
RELEASE_PKGS = peer orderer light

pkgmap.peer           := $(PKGNAME)/peer
pkgmap.light          := $(PKGNAME)/light
pkgmap.orderer        := $(PKGNAME)/orderer

ifneq ($(shell uname),Darwin)
DOCKER_RUN_FLAGS=--user=$(shell id -u)
endif

ifeq ($(shell uname -m),s390x)
ifneq ($(shell id -u),0)
DOCKER_RUN_FLAGS+=-v /etc/passwd:/etc/passwd:ro
endif
endif

ifneq ($(http_proxy),)
DOCKER_BUILD_FLAGS+=--build-arg 'http_proxy=$(http_proxy)'
DOCKER_RUN_FLAGS+=-e 'http_proxy=$(http_proxy)'
endif
ifneq ($(https_proxy),)
DOCKER_BUILD_FLAGS+=--build-arg 'https_proxy=$(https_proxy)'
DOCKER_RUN_FLAGS+=-e 'https_proxy=$(https_proxy)'
endif
ifneq ($(HTTP_PROXY),)
DOCKER_BUILD_FLAGS+=--build-arg 'HTTP_PROXY=$(HTTP_PROXY)'
DOCKER_RUN_FLAGS+=-e 'HTTP_PROXY=$(HTTP_PROXY)'
endif
ifneq ($(HTTPS_PROXY),)
DOCKER_BUILD_FLAGS+=--build-arg 'HTTPS_PROXY=$(HTTPS_PROXY)'
DOCKER_RUN_FLAGS+=-e 'HTTPS_PROXY=$(HTTPS_PROXY)'
endif
ifneq ($(no_proxy),)
DOCKER_BUILD_FLAGS+=--build-arg 'no_proxy=$(no_proxy)'
DOCKER_RUN_FLAGS+=-e 'no_proxy=$(no_proxy)'
endif
ifneq ($(NO_PROXY),)
DOCKER_BUILD_FLAGS+=--build-arg 'NO_PROXY=$(NO_PROXY)'
DOCKER_RUN_FLAGS+=-e 'NO_PROXY=$(NO_PROXY)'
endif

DRUN = docker run -i --rm $(DOCKER_RUN_FLAGS) \
	-v $(abspath .):/go/src/$(PKGNAME) \
	-w /go/src/$(PKGNAME)

DBUILD = docker build $(DOCKER_BUILD_FLAGS)

BASE_DOCKER_NS ?= rongzer
BASE_DOCKER_TAG=1.13.3

DOCKER_NS ?= rongzer
DOCKER_TAG=$(GIT_VERSION)

DOCKER_GO_LDFLAGS += $(GO_LDFLAGS)
DOCKER_GO_LDFLAGS += -linkmode external -extldflags '-lpthread'

#
# What is a .dummy file?
#
# Make is designed to work with files.  It uses the presence (or lack thereof)
# and timestamps of files when deciding if a given target needs to be rebuilt.
# Docker containers throw a wrench into the works because the output of docker
# builds do not translate into standard files that makefile rules can evaluate.
# Therefore, we have to fake it.  We do this by constructioning our rules such
# as
#       my-docker-target/.dummy:
#              docker build ...
#              touch $@
#
# If the docker-build succeeds, the touch operation creates/updates the .dummy
# file.  If it fails, the touch command never runs.  This means the .dummy
# file follows relatively 1:1 with the underlying container.
#
# This isn't perfect, however.  For instance, someone could delete a docker
# container using docker-rmi outside of the build, and make would be fooled
# into thinking the dependency is statisfied when it really isn't.  This is
# our closest approximation we can come up with.
#
# As an aside, also note that we incorporate the version number in the .dummy
# file to differentiate different tags to fix FAB-1145
#
DUMMY = .dummy-$(DOCKER_TAG)

all: docker

docker: $(patsubst %,build/image/%/$(DUMMY), $(IMAGES)) kafka zookeeper

.PHONY: base
base:
	cd images/buildenv && ./build.sh
	cd images/runenv && ./build.sh

.PHONY: peer
peer: build/image/peer/$(DUMMY)

.PHONY: light
light: build/image/light/$(DUMMY)

.PHONY: orderer
orderer: build/image/orderer/$(DUMMY)

.PHONY: javaenv
javaenv: build/image/javaenv/$(DUMMY)

.PHONY: kafka
kafka:
	cd images/kafka && ./build.sh

.PHONY: zookeeper
zookeeper:
	cd images/zookeeper && ./build.sh

.PHONY: test
test: build/test
# 各个模块的test用例输出分别写到build/test的各个子目录下
build/test: build/test/html \
            build/test/common_ledger_util_rocksdbhelper \
            build/test/peer_ledger_kvledger \
            build/test/peer_ledger_kvledger_txmgmt_statedb_stateleveldb \
            build/test/peer_gossip_state \
            build/test/peer_gossip_light \
            build/test/orderer \
            build/test/peer		\
            build/test/bccsp_crypto_util

# build/test/html用于存放所有的代码覆盖率html文件
build/test/html:
	@mkdir -p $@
	@cp setup/code-coverage-index.html $@

# orderer模块测试用例
build/test/orderer:
	@mkdir -p $@
	@echo "go test orderer"
	@export GO111MODULE=off
	@go test github.com/rongzer/blockchain/orderer/filters -v -coverprofile=$@/1.out 2>&1 | go-junit-report > $@/Test1.xml
	@go test github.com/rongzer/blockchain/orderer/filters/filter -v -coverprofile=$@/2.out 2>&1 | go-junit-report > $@/Test2.xml
	@go test github.com/rongzer/blockchain/orderer/endorse -v -coverprofile=$@/3.out 2>&1 | go-junit-report > $@/Test3.xml
	@go test github.com/rongzer/blockchain/orderer/ledger -v -coverprofile=$@/4.out 2>&1 | go-junit-report > $@/Test4.xml
	@go test github.com/rongzer/blockchain/orderer/localconfig -v -coverprofile=$@/5.out 2>&1 | go-junit-report > $@/Test5.xml
	@cd $@ && cat 1.out > c.out && cat 2.out|tail -n +2 >> c.out && cat 3.out|tail -n +2 >> c.out && cat 4.out|tail -n +2 >> c.out && cat 5.out|tail -n +2 >> c.out
	@go tool cover -html=$@/c.out -o build/test/html/orderer.html
	@sed -i 'N;8a\<a href="orderer.html">orderer</a>' build/test/html/code-coverage-index.html
# 	@gocover-cobertura < $@/c.out > $@/coverage.xml

# peer模块测试用例
build/test/peer:
	@mkdir -p $@
	@echo "go test peer"
	@export GO111MODULE=off
	@go test github.com/rongzer/blockchain/peer/chain -v -coverprofile=$@/1.out 2>&1 | go-junit-report > $@/Test1.xml
	@go test github.com/rongzer/blockchain/peer/endorser -ldflags="-X 'github.com/rongzer/blockchain/common/metadata.Version=1.0.0'" -v -coverprofile=$@/2.out 2>&1 | go-junit-report > $@/Test2.xml
	@go test github.com/rongzer/blockchain/peer/events/producer -v -coverprofile=$@/3.out 2>&1 | go-junit-report > $@/Test3.xml
	@go test github.com/rongzer/blockchain/peer/scc/ -v -coverprofile=$@/4.out 2>&1 | go-junit-report > $@/Test4.xml
	@go test github.com/rongzer/blockchain/peer/scc/cscc -v -coverprofile=$@/5.out 2>&1 | go-junit-report > $@/Test5.xml
	@go test github.com/rongzer/blockchain/peer/scc/escc -v -coverprofile=$@/6.out 2>&1 | go-junit-report > $@/Test6.xml
	@go test github.com/rongzer/blockchain/peer/scc/lscc -v -coverprofile=$@/7.out 2>&1 | go-junit-report > $@/Test7.xml
	@go test github.com/rongzer/blockchain/peer/scc/qscc -v -coverprofile=$@/8.out 2>&1 | go-junit-report > $@/Test8.xml
	@go test github.com/rongzer/blockchain/peer/scc/rbcapproval -v -coverprofile=$@/9.out 2>&1 | go-junit-report > $@/Test9.xml
	@go test github.com/rongzer/blockchain/peer/scc/rbcmodel -v -coverprofile=$@/10.out 2>&1 | go-junit-report > $@/Test10.xml
	@go test github.com/rongzer/blockchain/peer/scc/rbctoken -v -coverprofile=$@/11.out 2>&1 | go-junit-report > $@/Test11.xml
	@go test github.com/rongzer/blockchain/peer/scc/vscc -v -coverprofile=$@/12.out 2>&1 | go-junit-report > $@/Test12.xml
	@cd $@ && cat 1.out > c.out && cat 2.out|tail -n +2 >> c.out && cat 3.out|tail -n +2 >> c.out && cat 4.out|tail -n +2 >> c.out && cat 5.out|tail -n +2 && cat 6.out > c.out && cat 7.out|tail -n +2 >> c.out && cat 8.out|tail -n +2 >> c.out && cat 9.out|tail -n +2 >> c.out && cat 10.out|tail -n +2 && cat 11.out|tail -n +2 >> c.out && cat 12.out|tail -n +2 >> c.out
	@go tool cover -html=$@/c.out -o build/test/html/peer.html
	@sed -i 'N;8a\<a href="orderer.html">orderer</a>' build/test/html/code-coverage-index.html

# common下bccso, crypto, util包的测试
build/test/bccsp_crypto_util:
	@mkdir -p $@
	@echo "go test orderer"
	@export GO111MODULE=off
	@go test github.com/rongzer/blockchain/common/bccsp -v -coverprofile=$@/1.out 2>&1 | go-junit-report > $@/Test1.xml
	@go test github.com/rongzer/blockchain/common/bccsp/factory -v -coverprofile=$@/2.out 2>&1 | go-junit-report > $@/Test2.xml
	@go test github.com/rongzer/blockchain/common/bccsp/pkcs11 -v -coverprofile=$@/3.out 2>&1 | go-junit-report > $@/Test3.xml
	@go test github.com/rongzer/blockchain/common/bccsp/signer -v -coverprofile=$@/4.out 2>&1 | go-junit-report > $@/Test4.xml
	@go test github.com/rongzer/blockchain/common/bccsp/sw -v -coverprofile=$@/5.out 2>&1 | go-junit-report > $@/Test5.xml
	@go test github.com/rongzer/blockchain/common/bccsp/utils -v -coverprofile=$@/6.out 2>&1 | go-junit-report > $@/Test6.xml
	@go test github.com/rongzer/blockchain/common/crypto -v -coverprofile=$@/7.out 2>&1 | go-junit-report > $@/Test7.xml
	@go test github.com/rongzer/blockchain/common/util -v -coverprofile=$@/8.out 2>&1 | go-junit-report > $@/Test8.xml
	@cd $@ && cat 1.out > c.out && cat 2.out|tail -n +2 >> c.out && cat 3.out|tail -n +2 >> c.out && cat 4.out|tail -n +2 >> c.out && cat 5.out|tail -n +2 >> c.out && cat 6.out|tail -n +2 >> c.out && cat 7.out|tail -n +2 >> c.out && cat 8.out|tail -n +2 >> c.out
	@go tool cover -html=$@/c.out -o build/test/html/bccsp_crypto_util.html
	@sed -i 'N;8a\<a href="bccsp_crypto_util.html">bccsp_crypto_util</a>' build/test/html/code-coverage-index.html
# 	@gocover-cobertura < $@/c.out > $@/coverage.xml

# common/ledger/util/rocksdbhelper模块共4个测试用例：对rocksdb数据库的基本操作
build/test/common_ledger_util_rocksdbhelper:
	@mkdir -p $@
	@echo "go test common/ledger/util/rocksdbhelper"
	@export GO111MODULE=off
	@cd common/ledger/util/rocksdbhelper && go test -v rocksdb_helper_test.go rocksdb_helper.go -coverprofile=../../../../$@/1.out 2>&1 | go-junit-report > ../../../../$@/Test1.xml
	@cd common/ledger/util/rocksdbhelper && go test -v rocksdb_provider_test.go rocksdb_provider.go rocksdb_helper.go -coverprofile=../../../../$@/2.out 2>&1 | go-junit-report > ../../../../$@/Test2.xml
	@cd $@ && cat 1.out > c.out && cat 2.out|tail -n +2 >> c.out
	@go tool cover -html=$@/c.out -o build/test/html/common_ledger_util_rocksdbhelper.html
	@sed -i 'N;8a\<a href="common_ledger_util_rocksdbhelper.html">common_ledger_util_rocksdbhelper</a>' build/test/html/code-coverage-index.html
# 	@gocover-cobertura < $@/c.out > $@/coverage.xml

# peer/ledger/kvledger/txmgmt/statedb/stateleveldb模块
build/test/peer_ledger_kvledger_txmgmt_statedb_stateleveldb:
	@mkdir -p $@
	@echo "go test peer/ledger/kvledger/txmgmt/statedb/stateleveldb"
	@export GO111MODULE=off
	@cd peer/ledger/kvledger/txmgmt/statedb/stateleveldb && go test -v stateleveldb_test.go stateleveldb.go -coverprofile=../../../../../../$@/c.out 2>&1 | go-junit-report > ../../../../../../$@/Test1.xml
	@go tool cover -html=$@/c.out -o build/test/html/peer_ledger_kvledger_txmgmt_statedb_stateleveldb.html
	@sed -i 'N;8a\<a href="peer_ledger_kvledger_txmgmt_statedb_stateleveldb.html">peer_ledger_kvledger_txmgmt_statedb_stateleveldb</a>' build/test/html/code-coverage-index.html

# peer/ledger/kvledger模块共14个测试用例：包括账本的基本功能测试和边缘场景测试
build/test/peer_ledger_kvledger:
	@mkdir -p $@
	@echo "go test peer/ledger/kvledger"
	@export GO111MODULE=off
	@cd peer/ledger/kvledger && go test -v kv_ledger_test.go kv_ledger.go kv_ledger_provider.go recovery.go -test.run TestKVLedgerBlockStorage -coverprofile=../../../$@/1.out 2>&1 | go-junit-report > ../../../$@/Test1.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_test.go kv_ledger.go kv_ledger_provider.go recovery.go -test.run TestKVLedgerRecoveryNotUpdateStateDb_beforeFail -coverprofile=../../../$@/2.out 2>&1 | go-junit-report > ../../../$@/Test2.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_test.go kv_ledger.go kv_ledger_provider.go recovery.go -test.run TestKVLedgerRecoveryNotUpdateStateDb_afterFail -coverprofile=../../../$@/3.out 2>&1 | go-junit-report > ../../../$@/Test3.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_test.go kv_ledger.go kv_ledger_provider.go recovery.go -test.run TestKVLedgerRecoveryNotUpdateHistoryDb_beforeFail -coverprofile=../../../$@/4.out 2>&1 | go-junit-report > ../../../$@/Test4.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_test.go kv_ledger.go kv_ledger_provider.go recovery.go -test.run TestKVLedgerRecoveryNotUpdateHistoryDb_afterFail -coverprofile=../../../$@/5.out 2>&1 | go-junit-report > ../../../$@/Test5.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_test.go kv_ledger.go kv_ledger_provider.go recovery.go -test.run TestKVLedgerRecoveryRareScenario_beforeFail -coverprofile=../../../$@/6.out 2>&1 | go-junit-report > ../../../$@/Test6.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_test.go kv_ledger.go kv_ledger_provider.go recovery.go -test.run TestKVLedgerRecoveryRareScenario_afterFail -coverprofile=../../../$@/7.out 2>&1 | go-junit-report > ../../../$@/Test7.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_test.go kv_ledger.go kv_ledger_provider.go recovery.go -test.run TestKVLedgerIndexState -coverprofile=../../../$@/14.out 2>&1 | go-junit-report > ../../../$@/Test14.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_provider_test.go kv_ledger.go kv_ledger_provider.go recovery.go kv_ledger_test.go -test.run TestLedgerProvider -coverprofile=../../../$@/8.out 2>&1 | go-junit-report > ../../../$@/Test8.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_provider_test.go kv_ledger.go kv_ledger_provider.go recovery.go kv_ledger_test.go -test.run TestIdStoreRecovery_beforeFail -coverprofile=../../../$@/9.out 2>&1 | go-junit-report > ../../../$@/Test9.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_provider_test.go kv_ledger.go kv_ledger_provider.go recovery.go kv_ledger_test.go -test.run TestIdStoreRecovery_AfterFail -coverprofile=../../../$@/10.out 2>&1 | go-junit-report > ../../../$@/Test10.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_provider_test.go kv_ledger.go kv_ledger_provider.go recovery.go kv_ledger_test.go -test.run TestMultipleLedgerBasicRead -coverprofile=../../../$@/11.out 2>&1 | go-junit-report > ../../../$@/Test11.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_provider_test.go kv_ledger.go kv_ledger_provider.go recovery.go kv_ledger_test.go -test.run TestMultipleLedgerBasicWrite -coverprofile=../../../$@/12.out 2>&1 | go-junit-report > ../../../$@/Test12.xml
	@cd peer/ledger/kvledger && go test -v kv_ledger_provider_test.go kv_ledger.go kv_ledger_provider.go recovery.go kv_ledger_test.go -test.run TestLedgerBackup -coverprofile=../../../$@/13.out 2>&1 | go-junit-report > ../../../$@/Test13.xml
	@cd $@ && cat 1.out > c.out && cat 2.out|tail -n +2 >> c.out && cat 3.out|tail -n +2 >> c.out && cat 4.out|tail -n +2 >> c.out && cat 5.out|tail -n +2 >> c.out && cat 6.out|tail -n +2 >> c.out && cat 14.out|tail -n +2 >> c.out \
	 && cat 7.out|tail -n +2 >> c.out && cat 8.out|tail -n +2 >> c.out && cat 9.out|tail -n +2 >> c.out && cat 10.out|tail -n +2 >> c.out && cat 11.out|tail -n +2 >> c.out && cat 12.out|tail -n +2 >> c.out && cat 13.out|tail -n +2 >> c.out
	@go tool cover -html=$@/c.out -o build/test/html/peer_ledger_kvledger.html
	@sed -i 'N;8a\<a href="peer_ledger_kvledger.html">peer_ledger_kvledger</a>' build/test/html/code-coverage-index.html

# peer/gossip/light模块共6个测试用例：模拟轻量节点同步块时的gossip请求
build/test/peer_gossip_light:
	@mkdir -p $@
	@echo "go test peer/gossip/light"
	@export GO111MODULE=off
	@cd peer/gossip/light && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestLightGossipDirectMsg -coverprofile=../../../$@/1.out 2>&1 | go-junit-report > ../../../$@/Test1.xml
	@cd peer/gossip/light && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestLightGossipReception -coverprofile=../../../$@/2.out 2>&1 | go-junit-report > ../../../$@/Test2.xml
	@cd peer/gossip/light && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestLightGossipAccessControl -coverprofile=../../../$@/3.out 2>&1 | go-junit-report > ../../../$@/Test3.xml
	@cd peer/gossip/light && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestLightGossip_SendingManyMessages -coverprofile=../../../$@/4.out 2>&1 | go-junit-report > ../../../$@/Test4.xml
	@cd peer/gossip/light && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestLightGossip_LightWeightRequest -coverprofile=../../../$@/5.out 2>&1 | go-junit-report > ../../../$@/Test5.xml
	@cd peer/gossip/light && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestLightGossip_FailedLightWeightRequest -coverprofile=../../../$@/6.out 2>&1 | go-junit-report > ../../../$@/Test6.xml
	@cd $@ && cat 1.out > c.out && cat 2.out|tail -n +2 >> c.out && cat 3.out|tail -n +2 >> c.out && cat 4.out|tail -n +2 >> c.out && cat 5.out|tail -n +2 >> c.out \
	 && cat 6.out|tail -n +2 >> c.out
	@go tool cover -html=$@/c.out -o build/test/html/peer_gossip_light.html
	@sed -i 'N;8a\<a href="peer_gossip_light.html">peer_gossip_light</a>' build/test/html/code-coverage-index.html

# peer/gossip/state模块共7个测试用例：模拟普通节点同步块时的gossip请求
build/test/peer_gossip_state:
	@mkdir -p $@
	@echo "go test peer/gossip/state"
	@export GO111MODULE=off
	@cd peer/gossip/state && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestPeerGossipDirectMsg -coverprofile=../../../$@/1.out 2>&1 | go-junit-report > ../../../$@/Test1.xml
	@cd peer/gossip/state && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestPeerGossipReception -coverprofile=../../../$@/2.out 2>&1 | go-junit-report > ../../../$@/Test2.xml
	@cd peer/gossip/state && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestPeerGossipAccessControl -coverprofile=../../../$@/3.out 2>&1 | go-junit-report > ../../../$@/Test3.xml
	@cd peer/gossip/state && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestPeerGossip_SendingManyMessages -coverprofile=../../../$@/4.out 2>&1 | go-junit-report > ../../../$@/Test4.xml
	@cd peer/gossip/state && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestPeerGossip_StateRequest -coverprofile=../../../$@/5.out 2>&1 | go-junit-report > ../../../$@/Test5.xml
	@cd peer/gossip/state && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestPeerGossip_LightWeightRequest -coverprofile=../../../$@/6.out 2>&1 | go-junit-report > ../../../$@/Test6.xml
	@cd peer/gossip/state && go test -v state_test.go state.go payloads_buffer.go metastate.go -test.run TestPeerGossip_FailedLightWeightRequest -coverprofile=../../../$@/7.out 2>&1 | go-junit-report > ../../../$@/Test7.xml
	@cd $@ && cat 1.out > c.out && cat 2.out|tail -n +2 >> c.out && cat 3.out|tail -n +2 >> c.out && cat 4.out|tail -n +2 >> c.out && cat 5.out|tail -n +2 >> c.out \
	 && cat 6.out|tail -n +2 >> c.out && cat 7.out|tail -n +2 >> c.out
	@go tool cover -html=$@/c.out -o build/test/html/peer_gossip_state.html
	@sed -i 'N;8a\<a href="peer_gossip_state.html">peer_gossip_state</a>' build/test/html/code-coverage-index.html

# We (re)build a package within a docker context but persist the $GOPATH/pkg
# directory so that subsequent builds are faster
build/docker/bin/%: $(PROJECT_FILES)
	$(eval TARGET = ${patsubst build/docker/bin/%,%,${@}})
	@echo "Building $@"
	@echo "go build -o $@ -ldflags $(DOCKER_GO_LDFLAGS) $(pkgmap.$(@F))"
	@mkdir -p build/docker/bin build/docker/$(TARGET)/pkg
	@$(DRUN) \
		-v $(abspath build/docker/bin):/go/bin \
		-v $(abspath build/docker/$(TARGET)/pkg):/go/pkg \
        -e GO111MODULE=off \
		$(BASE_DOCKER_NS)/blockchain-buildenv:$(BASE_DOCKER_TAG) \
		go build -o $@ -ldflags "$(DOCKER_GO_LDFLAGS)" $(pkgmap.$(@F))
	@touch $@


changelog:
	./scripts/changelog.sh v$(PREV_VERSION) v$(BASE_VERSION)

# Both peer and peer-docker depend on ccenv and javaenv (all docker env images it supports).
# build/image/peer/$(DUMMY): build/image/ccenv/$(DUMMY) build/image/javaenv/$(DUMMY)

# payload definitions'
build/image/javaenv/payload:    build/javashim.tar.bz2 \
				build/protos.tar.bz2 \
				settings.gradle \
				setup/gradle-2.12-bin.zip \
				setup/protoc-gen-grpc-java \
				setup/apache-maven-3.3.9-bin.tar.gz
build/image/peer/payload:       build/docker/bin/peer \
				setup/signEnv.so \
				build/sampleconfig.tar.bz2 \
                images/peer/docker-entrypoint.sh
build/image/light/payload:       build/docker/bin/light \
				setup/signEnv.so \
				build/sampleconfig.tar.bz2 \
                images/light/docker-entrypoint.sh
build/image/orderer/payload:    build/docker/bin/orderer \
				build/sampleconfig.tar.bz2
build/image/kafka/payload:      images/kafka/docker-entrypoint.sh \
				images/kafka/kafka-run-class.sh \
				images/kafka/kafka_2.11-0.9.0.1.tgz

build/image/%/payload:
	mkdir -p $@
	cp $^ $@

.PRECIOUS: build/image/%/Dockerfile

build/image/%/Dockerfile: images/%/Dockerfile.in
	@cat $< \
		| sed -e 's!_BASE_NS_!$(BASE_DOCKER_NS)!g' \
		| sed -e 's!_NS_!$(DOCKER_NS)!g' \
		| sed -e 's!_BASE_TAG_!$(BASE_DOCKER_TAG)!g' \
		| sed -e 's!_TAG_!$(DOCKER_TAG)!g' \
		> $@
	@echo LABEL $(BASE_DOCKER_LABEL).version=$(PROJECT_VERSION) \\>>$@
	@echo "     " $(BASE_DOCKER_LABEL).base.version=$(BASEIMAGE_RELEASE)>>$@

build/image/%/$(DUMMY): Makefile build/image/%/payload build/image/%/Dockerfile
	$(eval TARGET = ${patsubst build/image/%/$(DUMMY),%,${@}})
	@echo "Building docker $(TARGET)-image"
	$(DBUILD) -t $(DOCKER_NS)/blockchain-$(TARGET) $(@D)
	docker tag $(DOCKER_NS)/blockchain-$(TARGET) $(DOCKER_NS)/blockchain-$(TARGET):$(DOCKER_TAG)
	@touch $@

build/sampleconfig.tar.bz2: $(shell find sampleconfig -type f)
	(cd sampleconfig && tar -jc *) > $@

build/javashim.tar.bz2: $(JAVASHIM_DEPS)
build/protos.tar.bz2: $(PROTOS)

build/%.tar.bz2:
	@echo "Creating $@"
	@tar -jc $^ > $@

%-docker-clean:
	$(eval TARGET = ${patsubst %-docker-clean,%,${@}})
	-docker images -q $(DOCKER_NS)/blockchain-$(TARGET):$(DOCKER_TAG) | xargs -I '{}' docker rmi -f '{}'
    -docker images -q $(DOCKER_NS)/blockchain-$(TARGET):latest | xargs -I '{}' docker rmi -f '{}'
	-@rm -rf build/image/$(TARGET) ||:

docker-clean: $(patsubst %,%-docker-clean, $(IMAGES))

.PHONY: clean
clean: docker-clean
	-@rm -rf build ||:

.PHONY: cleanPeer
cleanPeer: peer-docker-clean
	-@rm -rf build/docker/peer build/docker/bin/peer build/image/peer

.PHONY: cleanLight
cleanLight: light-docker-clean
	-@rm -rf build/docker/light build/docker/bin/light build/image/light

.PHONY: cleanOrderer
cleanOrderer: orderer-docker-clean
	-@rm -rf build/docker/orderer build/docker/bin/orderer build/image/orderer

.PHONY: cleanJavaenv
cleanJavaenv: javaenv-docker-clean
	-@rm -rf build/image/javaenv

.PHONY: cleanTest
cleanTest:
	@rm -rf build/