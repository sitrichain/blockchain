package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/tool/configtxlator/sanitycheck"
	"github.com/rongzer/blockchain/tool/configtxlator/update"
)

func fieldBytes(fieldName string, r *http.Request) ([]byte, error) {
	fieldFile, _, err := r.FormFile(fieldName)
	if err != nil {
		return nil, err
	}
	defer fieldFile.Close()

	return ioutil.ReadAll(fieldFile)
}

func fieldConfigProto(fieldName string, r *http.Request) (*cb.Config, error) {
	fieldBytes, err := fieldBytes(fieldName, r)
	if err != nil {
		return nil, fmt.Errorf("error reading field bytes: %s", err)
	}

	config := &cb.Config{}
	if err = config.Unmarshal(fieldBytes); err != nil {
		return nil, fmt.Errorf("error unmarshaling field bytes: %s", err)
	}

	return config, nil
}

func ComputeUpdateFromConfigs(w http.ResponseWriter, r *http.Request) {
	originalConfig, err := fieldConfigProto("original", r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error with field 'original': %s\n", err)
		return
	}

	updatedConfig, err := fieldConfigProto("updated", r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error with field 'updated': %s\n", err)
		return
	}

	configUpdate, err := update.Compute(originalConfig, updatedConfig)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error computing update: %s\n", err)
		return
	}

	configUpdate.ChannelId = r.FormValue("channel")

	encoded, err := configUpdate.Marshal()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error marshaling config update: %s\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(encoded)
}

func SanityCheckConfig(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}

	config := &cb.Config{}
	if err = config.Unmarshal(buf); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error unmarshaling data to common.Config': %s\n", err)
		return
	}

	fmt.Printf("Sanity checking %+v\n", config)
	sanityCheckMessages, err := sanitycheck.Check(config)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error performing sanity check: %s\n", err)
		return
	}

	resBytes, err := json.Marshal(sanityCheckMessages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error marshaling result to JSON: %s\n", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}
