package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/rongzer/blockchain/common/bccsp/factory"
	"github.com/rongzer/blockchain/common/config"
	mspconfig "github.com/rongzer/blockchain/common/config/msp"
	"github.com/rongzer/blockchain/common/configtx"
	"github.com/rongzer/blockchain/common/log"
	cb "github.com/rongzer/blockchain/protos/common"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
	genesisconfig "github.com/rongzer/blockchain/tool/configtxgen/localconfig"
	"github.com/rongzer/blockchain/tool/configtxgen/provisional"
)

func doOutputBlock(config *genesisconfig.Profile, channelID string, outputBlock string) error {
	// 针对共识名改为小写的问题, 强制指定sampleconsortium为大写
	if c, ok := config.Consortiums["sampleconsortium"]; ok {
		config.Consortiums["SampleConsortium"] = c
		delete(config.Consortiums, "sampleconsortium")
	}
	log.Logger.Infof("Consortium!!!!!!!!!! %v", config.Consortiums)

	pgen := provisional.New(config)
	log.Logger.Info("Generating genesis block")
	if config.Orderer == nil {
		return fmt.Errorf("config does not contain an Orderers section, necessary for all config blocks, aborting")
	}
	if config.Consortiums == nil {
		log.Logger.Warn("Genesis block does not contain a consortiums group definition.  This block cannot be used for orderer bootstrap.")
	}
	genesisBlock := pgen.GenesisBlockForChannel(channelID)
	log.Logger.Info("Writing genesis block")
	err := ioutil.WriteFile(outputBlock, utils.MarshalOrPanic(genesisBlock), 0644)
	if err != nil {
		return fmt.Errorf("Error writing genesis block: %s", err)
	}
	return nil
}

func doOutputChannelCreateTx(conf *genesisconfig.Profile, channelID string, outputChannelCreateTx string) error {
	log.Logger.Info("Generating new channel configtx")

	if conf.Application == nil {
		return fmt.Errorf("Cannot define a new channel with no Application section")
	}

	if conf.Consortium == "" {
		return fmt.Errorf("Cannot define a new channel with no Consortium value")
	}

	// XXX we ignore the non-application org names here, once the tool supports configuration updates
	// we should come up with a cleaner way to handle this, but leaving as is for the moment to not break
	// backwards compatibility
	var orgNames []string
	for _, org := range conf.Application.Organizations {
		orgNames = append(orgNames, org.Name)
	}
	creationTransaction, err := configtx.MakeChainCreationTransaction(channelID, conf.Consortium, nil, orgNames...)
	if err != nil {
		return fmt.Errorf("Error generating configtx: %s", err)
	}
	log.Logger.Info("Writing new channel tx")
	err = ioutil.WriteFile(outputChannelCreateTx, utils.MarshalOrPanic(creationTransaction), 0644)
	if err != nil {
		return fmt.Errorf("Error writing channel create tx: %s", err)
	}
	return nil
}

func doOutputAnchorPeersUpdate(conf *genesisconfig.Profile, channelID string, outputAnchorPeersUpdate string, asOrg string) error {
	log.Logger.Info("Generating anchor peer update")
	if asOrg == "" {
		return fmt.Errorf("Must specify an organization to update the anchor peer for")
	}

	if conf.Application == nil {
		return fmt.Errorf("Cannot update anchor peers without an application section")
	}

	var org *genesisconfig.Organization
	for _, iorg := range conf.Application.Organizations {
		if iorg.Name == asOrg {
			org = iorg
		}
	}

	if org == nil {
		return fmt.Errorf("No organization name matching: %s", asOrg)
	}

	anchorPeers := make([]*pb.AnchorPeer, len(org.AnchorPeers))
	for i, anchorPeer := range org.AnchorPeers {
		anchorPeers[i] = &pb.AnchorPeer{
			Host: anchorPeer.Host,
			Port: int32(anchorPeer.Port),
		}
	}

	configGroup := config.TemplateAnchorPeers(org.Name, anchorPeers)
	configUpdate := &cb.ConfigUpdate{
		ChannelId: channelID,
		WriteSet:  configGroup,
		ReadSet:   cb.NewConfigGroup(),
	}

	// Add all the existing config to the readset
	configUpdate.ReadSet.Groups[config.ApplicationGroupKey] = cb.NewConfigGroup()
	configUpdate.ReadSet.Groups[config.ApplicationGroupKey].Version = 1
	configUpdate.ReadSet.Groups[config.ApplicationGroupKey].Groups[org.Name] = cb.NewConfigGroup()
	configUpdate.ReadSet.Groups[config.ApplicationGroupKey].Groups[org.Name].Values[config.MSPKey] = &cb.ConfigValue{}
	configUpdate.ReadSet.Groups[config.ApplicationGroupKey].Groups[org.Name].Policies[mspconfig.ReadersPolicyKey] = &cb.ConfigPolicy{}
	configUpdate.ReadSet.Groups[config.ApplicationGroupKey].Groups[org.Name].Policies[mspconfig.WritersPolicyKey] = &cb.ConfigPolicy{}
	configUpdate.ReadSet.Groups[config.ApplicationGroupKey].Groups[org.Name].Policies[mspconfig.AdminsPolicyKey] = &cb.ConfigPolicy{}

	// Add all the existing at the same versions to the writeset
	configUpdate.WriteSet.Groups[config.ApplicationGroupKey].Version = 1
	configUpdate.WriteSet.Groups[config.ApplicationGroupKey].Groups[org.Name].Version = 1
	configUpdate.WriteSet.Groups[config.ApplicationGroupKey].Groups[org.Name].Values[config.MSPKey] = &cb.ConfigValue{}
	configUpdate.WriteSet.Groups[config.ApplicationGroupKey].Groups[org.Name].Policies[mspconfig.ReadersPolicyKey] = &cb.ConfigPolicy{}
	configUpdate.WriteSet.Groups[config.ApplicationGroupKey].Groups[org.Name].Policies[mspconfig.WritersPolicyKey] = &cb.ConfigPolicy{}
	configUpdate.WriteSet.Groups[config.ApplicationGroupKey].Groups[org.Name].Policies[mspconfig.AdminsPolicyKey] = &cb.ConfigPolicy{}

	configUpdateEnvelope := &cb.ConfigUpdateEnvelope{
		ConfigUpdate: utils.MarshalOrPanic(configUpdate),
	}

	update := &cb.Envelope{
		Payload: utils.MarshalOrPanic(&cb.Payload{
			Header: &cb.Header{
				ChannelHeader: utils.MarshalOrPanic(&cb.ChannelHeader{
					ChannelId: channelID,
					Type:      int32(cb.HeaderType_CONFIG_UPDATE),
				}),
			},
			Data: utils.MarshalOrPanic(configUpdateEnvelope),
		}),
	}

	log.Logger.Info("Writing anchor peer update")
	err := ioutil.WriteFile(outputAnchorPeersUpdate, utils.MarshalOrPanic(update), 0644)
	if err != nil {
		return fmt.Errorf("Error writing channel anchor peer update: %s", err)
	}
	return nil
}

func doInspectBlock(inspectBlock string) error {
	log.Logger.Info("Inspecting block")
	data, err := ioutil.ReadFile(inspectBlock)
	if err != nil {
		return fmt.Errorf("Could not read block %s", inspectBlock)
	}

	log.Logger.Info("Parsing genesis block")
	block := &cb.Block{}
	if err = block.Unmarshal(data); err != nil {
		return fmt.Errorf("Error unmarshaling block: %s", err)
	}

	ctx, err := utils.ExtractEnvelope(block, 0)
	if err != nil {
		return fmt.Errorf("Error retrieving configtx from block: %s", err)
	}

	payload, err := utils.UnmarshalPayload(ctx.Payload)
	if err != nil {
		return fmt.Errorf("Error extracting configtx payload: %s", err)
	}

	if payload.Header == nil {
		return fmt.Errorf("Config block did not contain header")
	}

	header, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return fmt.Errorf("Error unmarshaling channel header: %s", err)
	}

	if header.Type != int32(cb.HeaderType_CONFIG) {
		return fmt.Errorf("Bad header type: %d", header.Type)
	}

	configEnvelope, err := configtx.UnmarshalConfigEnvelope(payload.Data)
	if err != nil {
		return fmt.Errorf("Bad configuration envelope")
	}

	if configEnvelope.Config == nil {
		return fmt.Errorf("ConfigEnvelope contained no config")
	}

	configAsJSON, err := configGroupAsJSON(configEnvelope.Config.ChannelGroup)
	if err != nil {
		return err
	}

	fmt.Printf("Config for channel: %s at sequence %d\n", header.ChannelId, configEnvelope.Config.Sequence)
	fmt.Println(configAsJSON)

	return nil
}

func configGroupAsJSON(group *cb.ConfigGroup) (string, error) {
	configResult, err := configtx.NewConfigResult(group, configtx.NewInitializer())
	if err != nil {
		return "", fmt.Errorf("Error parsing config: %s", err)
	}

	buffer := &bytes.Buffer{}
	err = json.Indent(buffer, []byte(configResult.JSON()), "", "    ")
	if err != nil {
		return "", fmt.Errorf("Error in output JSON (usually a programming bug): %s", err)
	}
	return buffer.String(), nil
}

func doInspectChannelCreateTx(inspectChannelCreateTx string) error {
	log.Logger.Info("Inspecting transaction")
	data, err := ioutil.ReadFile(inspectChannelCreateTx)
	if err != nil {
		return fmt.Errorf("could not read channel create tx: %s", err)
	}

	log.Logger.Info("Parsing transaction")
	env, err := utils.UnmarshalEnvelope(data)
	if err != nil {
		return fmt.Errorf("Error unmarshaling envelope: %s", err)
	}

	payload, err := utils.UnmarshalPayload(env.Payload)
	if err != nil {
		return fmt.Errorf("Error extracting configtx payload: %s", err)
	}

	if payload.Header == nil {
		return fmt.Errorf("Config block did not contain header")
	}

	header, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return fmt.Errorf("Error unmarshaling channel header: %s", err)
	}

	if header.Type != int32(cb.HeaderType_CONFIG_UPDATE) {
		return fmt.Errorf("Bad header type: %d", header.Type)
	}

	configUpdateEnvelope, err := configtx.UnmarshalConfigUpdateEnvelope(payload.Data)
	if err != nil {
		return fmt.Errorf("Bad ConfigUpdateEnvelope")
	}

	configUpdate, err := configtx.UnmarshalConfigUpdate(configUpdateEnvelope.ConfigUpdate)
	if err != nil {
		return fmt.Errorf("ConfigUpdateEnvelope contained no config")
	}

	if configUpdate.ChannelId != header.ChannelId {
		return fmt.Errorf("ConfigUpdateEnvelope was for different channel than envelope: %s vs %s", configUpdate.ChannelId, header.ChannelId)
	}

	fmt.Printf("\nChannel creation for channel: %s\n", header.ChannelId)
	fmt.Println()

	if configUpdate.ReadSet == nil {
		fmt.Println("Read Set: empty")
	} else {
		fmt.Println("Read Set:")
		readSetAsJSON, err := configGroupAsJSON(configUpdate.ReadSet)
		if err != nil {
			return err
		}
		fmt.Println(readSetAsJSON)
	}
	fmt.Println()

	if configUpdate.WriteSet == nil {
		return fmt.Errorf("Empty WriteSet")
	}

	fmt.Println("Write Set:")
	writeSetAsJSON, err := configGroupAsJSON(configUpdate.WriteSet)
	if err != nil {
		return err
	}
	fmt.Println(writeSetAsJSON)
	fmt.Println()

	readSetMap, err := configtx.MapConfig(configUpdate.ReadSet)
	if err != nil {
		return fmt.Errorf("Error mapping read set: %s", err)
	}
	writeSetMap, err := configtx.MapConfig(configUpdate.WriteSet)
	if err != nil {
		return fmt.Errorf("Error mapping write set: %s", err)
	}

	fmt.Println("Delta Set:")
	deltaSet := configtx.ComputeDeltaSet(readSetMap, writeSetMap)
	for key := range deltaSet {
		fmt.Println(key)
	}
	fmt.Println()

	return nil
}

func main() {
	var outputBlock, outputChannelCreateTx, profile, channelID, inspectBlock, inspectChannelCreateTx, outputAnchorPeersUpdate, asOrg string

	flag.StringVar(&outputBlock, "outputBlock", "", "The path to write the genesis block to (if set)")
	flag.StringVar(&channelID, "channelID", provisional.TestChainID, "The channel ID to use in the configtx")
	flag.StringVar(&outputChannelCreateTx, "outputCreateChannelTx", "", "The path to write a channel creation configtx to (if set)")
	flag.StringVar(&profile, "profile", genesisconfig.SampleInsecureProfile, "The profile from configtx.yaml to use for generation.")
	flag.StringVar(&inspectBlock, "inspectBlock", "", "Prints the configuration contained in the block at the specified path")
	flag.StringVar(&inspectChannelCreateTx, "inspectChannelCreateTx", "", "Prints the configuration contained in the transaction at the specified path")
	flag.StringVar(&outputAnchorPeersUpdate, "outputAnchorPeersUpdate", "", "Creates an config update to update an anchor peer (works only with the default channel creation, and only for the first update)")
	flag.StringVar(&asOrg, "asOrg", "", "Performs the config generation as a particular organization (by name), only including values in the write set that org (likely) has privilege to set")

	flag.Parse()

	// don't need to panic when running via command line
	defer func() {
		if err := recover(); err != nil {
			if strings.Contains(fmt.Sprint(err), "Error reading configuration: Unsupported Config Type") {
				log.Logger.Error("Could not find configtx.yaml. " +
					"Please make sure that BLOCKCHAIN_CFG_PATH is set to a path " +
					"which contains configtx.yaml")
			}
			os.Exit(1)
		}
	}()

	log.Logger.Info("Loading configuration")
	factory.InitFactories(nil)
	conf := genesisconfig.Load(profile)

	if outputBlock != "" {
		if err := doOutputBlock(conf, channelID, outputBlock); err != nil {
			log.Logger.Fatalf("Error on outputBlock: %s", err)
		}
	}

	if outputChannelCreateTx != "" {
		if err := doOutputChannelCreateTx(conf, channelID, outputChannelCreateTx); err != nil {
			log.Logger.Fatalf("Error on outputChannelCreateTx: %s", err)
		}
	}

	if inspectBlock != "" {
		if err := doInspectBlock(inspectBlock); err != nil {
			log.Logger.Fatalf("Error on inspectBlock: %s", err)
		}
	}

	if inspectChannelCreateTx != "" {
		if err := doInspectChannelCreateTx(inspectChannelCreateTx); err != nil {
			log.Logger.Fatalf("Error on inspectChannelCreateTx: %s", err)
		}
	}

	if outputAnchorPeersUpdate != "" {
		if err := doOutputAnchorPeersUpdate(conf, channelID, outputAnchorPeersUpdate, asOrg); err != nil {
			log.Logger.Fatalf("Error on inspectChannelCreateTx: %s", err)
		}
	}
}
