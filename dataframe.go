package netdataframe

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
)

// head -c 8 /dev/urandom | hexdump -C
var dataFrameMagicWord = [...]byte{ 0xaa, 0xde, 0x07, 0xa6, 0x02, 0x57, 0xc2, 0xd0 }

type  DataFrame interface {
	// Returns true when data frame is fully captured
	IsFullFrame() bool
	// Decodes captured data frame
	Decode(receiver interface{}) error
	// Capture data stream into a data frame
	Capture(buf []byte)
	// Returns data frame
	GetFrame() []byte
	// Resets frame capturing returning captured before data
	Flush() []byte
	// Converts data into a frame using gob encoding
	ToFrame(data interface{}) []byte
}

type dataFrame struct {
	buf      []byte
	tail     int
	tailbuf  []byte
	isFull   bool
	framelen int
}

func NewDataFrame() *dataFrame {
	df := &dataFrame{}
	return df
}

func (f *dataFrame) IsFullFrame() bool {
	return f.isFull
}

func (f *dataFrame) Capture(data []byte) {
	if f.isFull {
		panic("data frame is full")
	} else if f.tail == 0 && f.probe(data) {
		// Calculate frame length and allocate data buffer
		mwl := len(dataFrameMagicWord)
		r := bytes.NewReader(data[mwl:2*mwl])
		var fl uint64
		binary.Read(r, binary.BigEndian, &fl)
		f.buf = make([]byte, fl)
		f.framelen = int(fl)

		// Copy the rest of data
		if len(f.tailbuf) > 0 {
			f.tail += copy(f.buf[0:], f.tailbuf[0:])
			f.tailbuf = nil
		}

		// Copy new data
		f.tail += copy(f.buf[f.tail:f.framelen], data[0+mwl+8:])
		if f.tail == f.framelen {
			f.isFull = true
		}
	} else if f.tail > 0 {
		f.tail += copy(f.buf[f.tail:], data[0:])
		if f.tail == f.framelen {
			f.isFull = true
			rest := len(data) - f.framelen
			if rest > 0 {
				f.tailbuf = make([]byte, rest)
				copy(f.tailbuf[0:], data[f.tail:])
			}
		}
	} else {
		f.buf = data
	}
}

func (f *dataFrame) GetFrame() []byte {
	out := make([]byte, len(f.buf))
	copy(out[0:], f.buf[0:])
	f.isFull = false
	return out
}

func (f *dataFrame) Decode(receiver interface{}) error {
	if ! f.isFull {
		panic("trying to decode incomplete data frame")
	}
	buf := bytes.NewBuffer(f.GetFrame())
	dec := gob.NewDecoder(buf)
	err := dec.Decode(receiver)
	return err
}

func (f *dataFrame) Flush() []byte {
	out := make([]byte, len(f.buf))
	copy(out[0:], f.buf[0:])
	f.tail = 0
	f.tailbuf = nil
	f.isFull = false
	return out
}

func (f *dataFrame) ToFrame(data interface{}) ([]byte, error) {
	var frame bytes.Buffer
	// Encode frame data
	enc := gob.NewEncoder(&frame)
	err := enc.Encode(data)
	if err != nil { return nil, err }

	// Write frame length in big endian order
	framelen := uint64(frame.Len())
	lbuf := new(bytes.Buffer)
	err = binary.Write(lbuf, binary.BigEndian, framelen)
	if err != nil { return nil, err }

	// Compose data stream
	mwl := len(dataFrameMagicWord)
	buf := make([]byte, mwl + lbuf.Len() + frame.Len())
	copy(buf[0:mwl], dataFrameMagicWord[:]) // data frame magic word
	copy(buf[mwl:], lbuf.Bytes()) // length
	copy(buf[mwl+lbuf.Len():], frame.Bytes()) // serialized data structure
	return buf, nil
}

func (f *dataFrame) probe(buf []byte) bool {
	if len(buf) < len(dataFrameMagicWord) { return false }

	for i := 0; i < len(dataFrameMagicWord); i++ {
		if buf[i] != dataFrameMagicWord[i] { return false }
	}

	return true
}
