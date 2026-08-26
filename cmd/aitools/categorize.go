package aitools

import (
	"errors"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// classifyInstallError maps an install failure to a telemetry error category.
// It classifies via typed/sentinel errors only (never error-message strings),
// so upstream wording can change without breaking the categories. A nil error
// is TYPE_UNSPECIFIED; anything unrecognized is UNCATEGORIZED so a newer failure
// mode is still counted rather than dropped.
func classifyInstallError(err error) protos.AitoolsErrorCategory {
	if err == nil {
		return protos.AitoolsErrorCategoryUnspecified
	}

	if blocked, ok := errors.AsType[*installer.BlockedError](err); ok {
		return blockedErrorCategory(blocked)
	}
	switch {
	case errors.Is(err, installer.ErrSkillNotFound):
		return protos.AitoolsErrorCategorySkillNotFound
	case errors.Is(err, installer.ErrVersionIncompatible):
		return protos.AitoolsErrorCategoryVersionIncompatible
	default:
		return protos.AitoolsErrorCategoryUncategorized
	}
}

// blockedErrorCategory maps a per-agent plugin BlockedError reason to its
// category. An unrecognized reason (e.g. ReasonNoPlugin, which callers filter
// out before install) is UNCATEGORIZED.
func blockedErrorCategory(e *installer.BlockedError) protos.AitoolsErrorCategory {
	switch e.Reason {
	case installer.ReasonCLINotOnPath:
		return protos.AitoolsErrorCategoryCLINotOnPath
	case installer.ReasonInstallFailed:
		return protos.AitoolsErrorCategoryPluginInstallFailed
	default:
		return protos.AitoolsErrorCategoryUncategorized
	}
}
