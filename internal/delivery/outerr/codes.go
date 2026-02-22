package outerr

// Stable, client-facing codes (use in frontend/mobile; avoid renaming).
const (
	CodeInternalError         = "INTERNAL_ERROR"
	CodeBadRequest            = "BAD_REQUEST"
	CodeNotFound              = "NOT_FOUND"
	CodeConflict              = "CONFLICT"
	CodeNoChanges             = "NO_CHANGES"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeForbidden             = "FORBIDDEN"
	CodeTooLargeEntity        = "TOO_LARGE_ENTITY"
	CodeTooManyRequests       = "TOO_MANY_REQUESTS"
	CodeUnprocessableEntity   = "UNPROCESSABLE_ENTITY"
	CodeRequestEntityTooLarge = "REQUEST_ENTITY_TOO_LARGE"

	// External Service codes
	CodeExternalService = "EXTERNAL_SERVICE"

	// OTP-specific codes
	CodeOTPNotFound = "OTP_NOT_FOUND"
	CodeOTPExpired  = "OTP_EXPIRED"
	CodeOTPMismatch = "OTP_MISMATCH"
	CodeOTPTooMany  = "OTP_TOO_MANY_ATTEMPTS"

	// Wallet-specific codes
	CodeInsufficientFunds   = "INSUFFICIENT_FUNDS"
	CodeUsageLimitExhausted = "USAGE_LIMIT_EXHAUSTED"

	// Taro
	CodeQuestionViolation = "QUESTION_VIOLATION"

	// WS
	CodeBadEvent         = "BAD_EVENT"
	CodeJoinChatRequired = "JOIN_CHAT_REQUIRED"

	// Profile
	CodeMissingUserProfile = "MISSING_USER_PROFILE"
)
