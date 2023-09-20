module github.com/rongzer/blockchain

go 1.13

replace (
	github.com/golang/lint => golang.org/x/lint v0.0.0-20190930215403-16217165b5de
	gopkg.in/flyaways/pool.v1 => github.com/flyaways/pool v1.0.1
)

require (
	code.cloudfoundry.org/clock v1.0.0 // indirect
	github.com/Knetic/govaluate v3.0.0+incompatible
	github.com/Shopify/sarama v1.24.1
	github.com/arthurkiller/rollingwriter v1.1.2
	github.com/facebookgo/ensure v0.0.0-20160127193407-b4ab57deab51 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20150612182917-8dac2c3c4870 // indirect
	github.com/fsouza/go-dockerclient v1.5.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-version v1.2.0
	github.com/hyperledger/fabric v2.0.1+incompatible
	github.com/hyperledger/fabric-amcl v0.0.0-20200128223036-d1aa2665426a // indirect
	github.com/hyperledger/fabric-protos-go v0.0.0-20200124220212-e9cfc186ba7b
	github.com/json-iterator/go v1.1.8
	github.com/klauspost/compress v1.9.2
	github.com/looplab/fsm v0.1.0
	github.com/miekg/pkcs11 v1.0.3
	github.com/minio/sha256-simd v0.1.1
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/ginkgo v1.10.2 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/sykesm/zap-logfmt v0.0.3 // indirect
	github.com/syndtr/goleveldb v1.0.0
	github.com/tecbot/gorocksdb v0.0.0-20191019123150-400c56251341
	go.etcd.io/etcd v3.3.18+incompatible // indirect
	go.uber.org/atomic v1.5.0
	go.uber.org/ratelimit v0.1.0
	go.uber.org/zap v1.12.0
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/net v0.0.0-20191021144547-ec77196f6094
	golang.org/x/sys v0.0.0-20191025090151-53bf42e6b339 // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/grpc v1.26.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.2.5
)
