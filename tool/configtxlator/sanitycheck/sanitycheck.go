package sanitycheck

import (
	"fmt"

	"github.com/rongzer/blockchain/common/configtx"
	cb "github.com/rongzer/blockchain/protos/common"
	mspprotos "github.com/rongzer/blockchain/protos/msp"
	"github.com/rongzer/blockchain/protos/utils"
)

type Messages struct {
	GeneralErrors   []string          `json:"general_errors"`
	ElementWarnings []*ElementMessage `json:"element_errors"`
	ElementErrors   []*ElementMessage `json:"element_errors"`
}

type ElementMessage struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func Check(config *cb.Config) (*Messages, error) {
	envConfig, err := utils.CreateSignedEnvelope(cb.HeaderType_CONFIG, "sanitycheck", nil, &cb.ConfigEnvelope{Config: config}, 0, 0)
	if err != nil {
		return nil, err
	}

	result := &Messages{}

	cm, err := configtx.NewConfigManager(envConfig, configtx.NewInitializer(), nil)
	if err != nil {
		result.GeneralErrors = []string{err.Error()}
		return result, nil
	}

	// This should come from the MSP manager, but, for some reason
	// the MSP manager is not intialized if there are no orgs, so,
	// we collect this manually.
	mspMap := make(map[string]struct{})

	if ac, ok := cm.ApplicationConfig(); ok {
		for _, org := range ac.Organizations() {
			mspMap[org.MSPID()] = struct{}{}
		}
	}

	if oc, ok := cm.OrdererConfig(); ok {
		for _, org := range oc.Organizations() {
			mspMap[org.MSPID()] = struct{}{}
		}
	}

	policyWarnings, policyErrors := checkPolicyPrincipals(config.ChannelGroup, "", mspMap)

	result.ElementWarnings = policyWarnings
	result.ElementErrors = policyErrors

	return result, nil
}

func checkPolicyPrincipals(group *cb.ConfigGroup, basePath string, mspMap map[string]struct{}) (warnings []*ElementMessage, errors []*ElementMessage) {
	for policyName, configPolicy := range group.Policies {
		appendError := func(err string) {
			errors = append(errors, &ElementMessage{
				Path:    basePath + ".policies." + policyName,
				Message: err,
			})
		}

		appendWarning := func(err string) {
			warnings = append(errors, &ElementMessage{
				Path:    basePath + ".policies." + policyName,
				Message: err,
			})
		}

		if configPolicy.Policy == nil {
			appendError(fmt.Sprintf("no policy value set for %s", policyName))
			continue
		}

		if configPolicy.Policy.Type != int32(cb.Policy_SIGNATURE) {
			continue
		}
		spe := &cb.SignaturePolicyEnvelope{}
		if err := spe.Unmarshal(configPolicy.Policy.Value); err != nil {
			appendError(fmt.Sprintf("error unmarshaling policy value to SignaturePolicyEnvelope: %s", err))
			continue
		}

		for i, identity := range spe.Identities {
			var mspID string
			switch identity.PrincipalClassification {
			case mspprotos.MSPPrincipal_ROLE:
				role := &mspprotos.MSPRole{}
				if err := role.Unmarshal(identity.Principal); err != nil {
					appendError(fmt.Sprintf("value of identities array at index %d is of type ROLE, but could not be unmarshaled to msp.MSPRole: %s", i, err))
					continue
				}
				mspID = role.MspIdentifier
			case mspprotos.MSPPrincipal_ORGANIZATION_UNIT:
				ou := &mspprotos.OrganizationUnit{}
				if err := ou.Unmarshal(identity.Principal); err != nil {
					appendError(fmt.Sprintf("value of identities array at index %d is of type ORGANIZATION_UNIT, but could not be unmarshaled to msp.OrganizationUnit: %s", i, err))
					continue
				}
				mspID = ou.MspIdentifier
			default:
				continue
			}

			_, ok := mspMap[mspID]
			if !ok {
				appendWarning(fmt.Sprintf("identity principal at index %d refers to MSP ID '%s', which is not an MSP in the network", i, mspID))
			}
		}
	}

	for subGroupName, subGroup := range group.Groups {
		subWarnings, subErrors := checkPolicyPrincipals(subGroup, basePath+".groups."+subGroupName, mspMap)
		warnings = append(warnings, subWarnings...)
		errors = append(errors, subErrors...)
	}
	return
}
