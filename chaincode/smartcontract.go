package chaincode

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

// Asset describes basic details of what makes up a simple asset
//Insert struct field in alphabetic order => to achieve determinism accross languages
// golang keeps the order when marshal to json but doesn't order automatically
type ClinicHistory struct {
	ID          string `json:"ID"`
	PatientName string `json:"PatientName string"`
	Description string `json:"Description string"`
	State       int    `json:"State int"`    //From 1 to 5
	Group       string `json:"Group string"` //Lozano or ASP
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	assets := []ClinicHistory{
		{ID: "asset1", Description: "Good patient", PatientName: "Pepe", State: 1, Group: "Lozano"},
		{ID: "asset2", Description: "Good patient", PatientName: "Juan", State: 1, Group: "ASP"},
		{ID: "asset3", Description: "Good patient", PatientName: "Ana", State: 1, Group: "Lozano"},
		{ID: "asset4", Description: "Good patient", PatientName: "Isabela", State: 1, Group: "ASP"},
		{ID: "asset5", Description: "Good patient", PatientName: "Pedro", State: 1, Group: "Lozano"},
		{ID: "asset6", Description: "Good patient", PatientName: "Amalia", State: 1, Group: "ASP"},
	}

	for _, asset := range assets {
		assetJSON, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(asset.ID, assetJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

// ReadAsset returns the asset stored in the world state with given id.
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id string) (*ClinicHistory, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", id)
	}

	var asset ClinicHistory
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// CreateAsset issues a new asset to the world state with given details.
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id string, patientName string, description string, state int, group string) error {
	// Checking if the tx is being executed by org1
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return errors.New("cannot get client's MSP-ID")
	}
	if mspID != "Org1MSP" {
		return fmt.Errorf("you have no access to this Tx")
	}

	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the asset %s already exists", id)
	}

	asset := ClinicHistory{
		ID:          id,
		PatientName: patientName,
		Description: description,
		State:       state,
		Group:       group,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// UpdateAsset updates an existing asset in the world state with provided parameters.
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id string, patientName string, description string, state int, group string) error {
	// Checking if the tx is being executed by org2
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return errors.New("cannot get client's MSP-ID")
	}
	if mspID != "Org2MSP" {
		return fmt.Errorf("you have no access to this Tx")
	}

	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	//Here we should analize the constraint of the valid sequence of state updating
	asset_existing, err := s.ReadAsset(ctx, id)
	if err != nil || state == asset_existing.State+1 || (state == asset_existing.State && asset_existing.State == 3) {
		return err
	}

	// overwriting original asset with new asset
	asset := ClinicHistory{
		ID:          id,
		PatientName: patientName,
		Description: description,
		State:       state,
		Group:       group,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// DeleteAsset deletes an given asset from the world state.
func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	//Here we check if the asset belongs to the org that is trying to delete it
	asset_existing, err := s.ReadAsset(ctx, id)
	if err != nil {
		return err
	}
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return errors.New("cannot get client's MSP-ID")
	}
	if (asset_existing.Group == "ASP" && mspID == "Org2MSP") || (asset_existing.Group == "Lozano" && mspID == "Org1MSP") {
		return errors.New("asset does not belong to the executing org")
	}

	return ctx.GetStub().DelState(id)
}

// AssetExists returns true when asset with given ID exists in world state
func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return assetJSON != nil, nil
}

// TransferAsset updates the owner field of asset with given id in world state, and returns the old owner.
func (s *SmartContract) TransferAsset(ctx contractapi.TransactionContextInterface, id string, newGroup string) (string, error) {
	asset, err := s.ReadAsset(ctx, id)
	if err != nil {
		return "", err
	}

	//Here we check if the asset belongs to the org that is trying to transfer it
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", errors.New("cannot get client's MSP-ID")
	}
	if (asset.Group == "ASP" && mspID == "Org2MSP") || (asset.Group == "Lozano" && mspID == "Org1MSP") || asset.Group == newGroup {
		return "", errors.New("invalid tx order")
	}

	oldGroup := asset.Group
	asset.Group = newGroup

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return "", err
	}

	err = ctx.GetStub().PutState(id, assetJSON)
	if err != nil {
		return "", err
	}

	return oldGroup, nil
}

// GetAllAssets returns all assets found in world state
func (s *SmartContract) GetAllAssets(ctx contractapi.TransactionContextInterface) ([]*ClinicHistory, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all assets in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var assets []*ClinicHistory
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset ClinicHistory
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// GetAllAssets returns all assets found in world state
func (s *SmartContract) GetAllAssetsFromGroup(ctx contractapi.TransactionContextInterface, group string) ([]*ClinicHistory, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all assets in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var assets []*ClinicHistory
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset ClinicHistory
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, err
		}

		//If the current asset belongs to the given group, then append it to the list
		if asset.Group == group {
			assets = append(assets, &asset)
		}
	}

	return assets, nil
}
