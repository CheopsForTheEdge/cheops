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

		// Run everything starting from the first command that's hasn't been run,
		// and everything after. It's ok because operations are
		// idempotent
		var add bool
		for _, unit := range d.Units {
			_, alreadyDone := existingReplies[unit.RequestId]

			// If the unit has already been run but we add everything, add it
			if alreadyDone && add {
				unitsToRun = append(unitsToRun, unit)

				// If the unit has not been run yet, we add it and set to add everything
			} else if !alreadyDone {
				unitsToRun = append(unitsToRun, unit)
				add = true
			}

		}

	case model.OperationsTypeNothing:
		return nil, fmt.Errorf("Resource can't be handled because it's consistency model doesn't allow it")
	}
	return unitsToRun, nil
}
