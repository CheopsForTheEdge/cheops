package replicator

import (
	"encoding/json"
	"fmt"

	"cheops.com/model"
)

// findUnitsToRun outputs the units to run based on the consistency model
// and whether they have already been run or not.
func findUnitsToRun(d model.ResourceDocument, existingReplies map[string]struct{}) ([]model.CrdtUnit, error) {
	if len(d.Units) == 0 {
		return nil, nil
	}

	configFile := d.Units[len(d.Units)-1].Command.Files["config.json"]
	var config model.ResourceConfig
	err := json.Unmarshal(configFile, &config)
	if err != nil {
		return nil, fmt.Errorf("Invalid config file: %v", err)
	}

	unitsToRun := make([]model.CrdtUnit, 0)

	switch config.OperationsType {
	case model.OperationsTypeCommutativeIdempotent:
		fallthrough
	case model.OperationsTypeCommutative:
		// Run everything that wasn't already run, even if it's earlier in the log
		// It's ok because operations are commutative
		for _, unit := range d.Units {
			_, alreadyDone := existingReplies[unit.RequestId]
			if alreadyDone {
				continue
			}
			unitsToRun = append(unitsToRun, unit)
		}

	case model.OperationsTypeIdempotent:
		// Run the last one, only if it wasn't already ran
		last := d.Units[len(d.Units)-1]
		_, alreadyRan := existingReplies[last.RequestId]
		if !alreadyRan {
			unitsToRun = append(unitsToRun, last)
		}

	case model.OperationsTypeNothing:
		return nil, fmt.Errorf("Resource can't be handled because it's consistency model doesn't allow it")
	}
	return unitsToRun, nil
}
