package inerr

import "errors"

// Routing engine (Valhalla) errors, grouped by what the caller should do
// about them. The client wraps one of these with the engine's own code and
// message: fmt.Errorf("%w: valhalla 444 ...", ErrRoutingNoMatch).
var (
	// ErrRoutingNoMatch: the engine found no road path for these points
	// (no road nearby, no connection, or its matcher failed). Splitting the
	// input into smaller parts may help; a retry of the same input won't.
	ErrRoutingNoMatch = errors.New("routing: no road path for the points")
	// ErrRoutingLimitExceeded: the request is over the engine's limits
	// (too many points, too long a path). Split the input.
	ErrRoutingLimitExceeded = errors.New("routing: request exceeds engine limits")
	// ErrRoutingBadRequest: the engine rejected the request as malformed.
	// A bug on our side; retrying won't help.
	ErrRoutingBadRequest = errors.New("routing: bad request")
	// ErrRoutingUnavailable: the engine couldn't be reached or failed
	// internally. Worth retrying later.
	ErrRoutingUnavailable = errors.New("routing: engine unavailable")
)
