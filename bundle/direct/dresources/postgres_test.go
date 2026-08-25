package dresources

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgresSpecUpdateMasks asserts that every field of every Postgres spec has
// an entry in the resource's update mask. Without this, a new field in the SDK
// would either be silently left out of update_mask or, worse, sent under a path
// the API does not know, failing the whole PATCH.
func TestPostgresSpecUpdateMasks(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		mask specUpdateMask
	}{
		{"postgres_projects", reflect.TypeFor[postgres.ProjectSpec](), projectSpecUpdateMask},
		{"postgres_branches", reflect.TypeFor[postgres.BranchSpec](), branchSpecUpdateMask},
		{"postgres_endpoints", reflect.TypeFor[postgres.EndpointSpec](), endpointSpecUpdateMask},
		{"postgres_roles", reflect.TypeFor[postgres.RoleRoleSpec](), roleSpecUpdateMask},
		{"postgres_databases", reflect.TypeFor[postgres.DatabaseDatabaseSpec](), databaseSpecUpdateMask},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			covered := map[string]bool{}
			err := structwalk.WalkType(tc.typ, func(path *structpath.PatternNode, typ reflect.Type, _ *reflect.StructField) bool {
				p := path.String()
				if p == "" {
					return true
				}
				if _, ok := tc.mask[p]; ok {
					covered[p] = true
					// The mask names this field, so it is a leaf as far as
					// update_mask is concerned: do not descend into it.
					return false
				}
				// Only a nested message may be absent from the mask; the walk
				// descends into it and the mask covers its fields instead.
				assert.Equal(t, reflect.Struct, derefType(typ).Kind(), "spec field %q has no entry in %s update mask", p, tc.name)
				return true
			})
			require.NoError(t, err)

			for p := range tc.mask {
				assert.Contains(t, covered, p, "%s update mask has entry %q, which is not a field of %s", tc.name, p, tc.typ.Name())
			}
		})
	}
}

func TestSpecUpdateMaskPaths(t *testing.T) {
	tests := []struct {
		name string
		spec any
		mask specUpdateMask
		want []string
	}{
		{
			name: "an element changing in place still masks the whole list",
			spec: &postgres.ProjectSpec{
				DisplayName: "p",
				CustomTags:  []postgres.ProjectCustomTag{{Key: "release_id", Value: "release-2"}},
			},
			mask: projectSpecUpdateMask,
			want: []string{"spec.custom_tags", "spec.display_name"},
		},
		{
			name: "fields the config leaves unset are not masked",
			spec: &postgres.ProjectSpec{DisplayName: "p"},
			mask: projectSpecUpdateMask,
			want: []string{"spec.display_name"},
		},
		{
			name: "an immutable field is left out of the mask",
			spec: &postgres.ProjectSpec{DisplayName: "p", PgVersion: 16},
			mask: projectSpecUpdateMask,
			want: []string{"spec.display_name"},
		},
		{
			name: "a partly populated nested message masks only the fields it sets",
			spec: &postgres.ProjectSpec{
				DefaultEndpointSettings: &postgres.ProjectDefaultEndpointSettings{AutoscalingLimitMinCu: 0.5},
			},
			mask: projectSpecUpdateMask,
			want: []string{"spec.default_endpoint_settings.autoscaling_limit_min_cu"},
		},
		{
			name: "oneof members collapse to the group the API accepts",
			spec: &postgres.BranchSpec{NoExpiry: true, IsProtected: true},
			mask: branchSpecUpdateMask,
			want: []string{"spec.expiration", "spec.is_protected"},
		},
		{
			name: "a map is masked whole",
			spec: &postgres.EndpointSpec{
				Settings: &postgres.EndpointSettings{PgSettings: map[string]string{"work_mem": "4MB"}},
			},
			mask: endpointSpecUpdateMask,
			want: []string{"spec.settings.pg_settings"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			paths, err := specUpdateMaskPaths(tc.spec, tc.mask)
			require.NoError(t, err)
			assert.Equal(t, tc.want, paths)
		})
	}
}

func TestSpecUpdateMaskPathsUnknownField(t *testing.T) {
	_, err := specUpdateMaskPaths(&postgres.DatabaseDatabaseSpec{Role: "r"}, specUpdateMask{})
	assert.EqualError(t, err, `no update_mask path known for spec field "role"`)
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
