package netdataframe

type Receiver interface {
	// Captures all available frames in buffer
	Capture(buf []byte) ([]*dataFrame, error)
	// Resets frame capturing returning captured before data
	Flush() []byte
	// Returns true if no data were captured
	IsEmpty() bool
}

type Sender interface {
	// Converts data into a frame using gob encoding
	ToFrame(data interface{}) (*dataFrame, error)
}

type DataFrame interface {
	// Decodes captured data frame using gob decoder
	Decode(receiver interface{}) error

	GetBytes() []byte
}

func NewReceiver() *receiver {
	recv := &receiver{}
	return recv
}
