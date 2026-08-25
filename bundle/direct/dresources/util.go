package dresources

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/retries"
)

type StateLifecycle struct {
	Started *bool `json:"started,omitempty"`
}

// This is copied from the retries package of the databricks-sdk-go. It should be made public,
// but for now, I'm copying it here.
func shouldRetry(err error) bool {
	if e, ok := errors.AsType[*retries.Err](err); ok {
		return !e.Halt
	}
	return false
}

// specUpdateMask maps a field path inside a resource spec to the path the API
// accepts for that field in update_mask. An empty target means the field cannot
// be set through update_mask at all; a change to it must be handled some other
// way (recreate_on_changes, ignore_local_changes). Keys name the leaves of the
// spec: a struct field is only a key once its own fields are not addressable
// individually, so repeated fields and maps appear as keys while nested
// messages do not.
type specUpdateMask map[string]string

// specUpdateMaskPaths returns the update_mask for a PATCH request whose body
// carries spec under "spec".
//
// The paths come from the body rather than from the plan's change list because
// the API validates the two against each other: every leaf under a masked path
// must be populated in the request, and a path the API does not know is
// rejected outright. In particular there is no path for an individual element
// of a repeated field or map entry, which is what the change list produces when
// a single element changes in place ("spec.custom_tags[0].value").
func specUpdateMaskPaths(spec any, mask specUpdateMask) ([]string, error) {
	raw, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}

	var paths []string
	if err := appendMaskPaths(&paths, body, "", mask); err != nil {
		return nil, err
	}

	slices.Sort(paths)
	return slices.Compact(paths), nil
}

// appendMaskPaths walks the request body and appends the mask path for every
// field it populates. Fields the mask does not name are nested messages the walk
// descends into; anything else means the mask is missing an entry.
func appendMaskPaths(paths *[]string, body map[string]any, prefix string, mask specUpdateMask) error {
	for field, value := range body {
		path := prefix + field
		target, ok := mask[path]
		if !ok {
			nested, isMessage := value.(map[string]any)
			if !isMessage {
				return fmt.Errorf("no update_mask path known for spec field %q", path)
			}
			if err := appendMaskPaths(paths, nested, path+".", mask); err != nil {
				return err
			}
			continue
		}
		if target != "" {
			*paths = append(*paths, "spec."+target)
		}
	}
	return nil
}

// hasSpecChanges reports whether the plan updates any field other than the given
// input-only ones. Those live in state but have no spec counterpart, so a change
// confined to them is a state-only refresh with nothing to PATCH.
func hasSpecChanges(changes Changes, inputOnly ...string) bool {
	for path, change := range changes {
		if change.Action == deployplan.Update && !slices.Contains(inputOnly, path) {
			return true
		}
	}
	return false
}
