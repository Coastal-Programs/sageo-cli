// Package recommendations defines the atomic "what to change" unit used by
// the recommendation engine, LLM drafting, and the PDF report.
//
// The underlying structs are declared in internal/state to allow State to
// embed Recommendation without creating an import cycle. This package
// re-exports those types via type aliases so callers can depend on
// "internal/recommendations" as the canonical API surface.
package recommendations

import "github.com/jakeschepis/sageo-cli/internal/state"

// Type aliases re-exported from internal/state. See the comment in
// internal/state/recommendations.go for the rationale.
type (
	ChangeType     = state.ChangeType
	Evidence       = state.Evidence
	Recommendation = state.Recommendation
	Forecast       = state.Forecast
	State          = state.State
)

// ChangeType constants re-exported from state so callers don't need to
// import the state package for enum values.
const (
	ChangeTitle             = state.ChangeTitle
	ChangeMeta              = state.ChangeMeta
	ChangeH1                = state.ChangeH1
	ChangeH2                = state.ChangeH2
	ChangeSchema            = state.ChangeSchema
	ChangeBody              = state.ChangeBody
	ChangeInternalLink      = state.ChangeInternalLink
	ChangeSpeed             = state.ChangeSpeed
	ChangeBacklink          = state.ChangeBacklink
	ChangeIndexability      = state.ChangeIndexability
	ChangeTLDR              = state.ChangeTLDR
	ChangeListFormat        = state.ChangeListFormat
	ChangeAuthorByline      = state.ChangeAuthorByline
	ChangeFreshness         = state.ChangeFreshness
	ChangeEntityConsistency = state.ChangeEntityConsistency
)
