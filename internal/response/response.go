package response

const (
	// StatusSuccess indicates all went well, and data is returned.
	StatusSuccess = "success"
)

// Response represents a generic JSend / Custom Envelope structure for successful API responses.
// Conforms to JSend (status, data) with optional custom envelope message support.
// Error and failure responses are handled separately via RFC 7807 (Problem Details).
type Response[T any] struct {
	Status  string `json:"status"`            // "success"
	Message string `json:"message,omitempty"` // Human-readable status message
	Data    T      `json:"data"`              // Payload data
}

// Success constructs a new JSend success envelope containing data and an optional message.
func Success[T any](data T, message ...string) *Response[T] {
	var msg string
	if len(message) > 0 {
		msg = message[0]
	}
	return &Response[T]{
		Status:  StatusSuccess,
		Message: msg,
		Data:    data,
	}
}
