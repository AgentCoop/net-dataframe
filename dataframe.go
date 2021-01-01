package netdataframe

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
)

// head -c 8 /dev/urandom | hexdump -C
var dataFrameMagicWord = [...]byte{ 0xaa, 0xde, 0x07, 0xa6, 0x02, 0x57, 0xc2, 0xd0 }

const (
	MagicWordLen = len(dataFrameMagicWord)
	DataFrameLen = 8
	ForewordLen = MagicWordLen + DataFrameLen
)

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
	ncap	int
	framelen int
}

func NewDataFrame() *dataFrame {
	df := &dataFrame{}
	return df
}

func (f *dataFrame) IsFullFrame() bool {
	return f.isFull
}

func (f *dataFrame) copy(data []byte) {
	n := copy(f.buf[f.tail:], data[0:])
	f.tail += n
	f.ncap += n
	if f.tail == f.framelen {
		rest := len(data) - n
		if rest > 0 {
			f.tailbuf = make([]byte, rest)
			copy(f.tailbuf[0:], data[n:])
			if len(f.tailbuf) > MagicWordLen && ! f.probe(f.tailbuf) {
				panic("tail")
			}
			//f.tail = 0
		}
		f.isFull = true
		f.tail = 0
	}
}

func (f *dataFrame) Capture(data []byte) {
	if f.isFull {
		panic("data frame is full")
	}

	// Capture one extra byte to grow tail by one
	if len(f.tailbuf) + len(data) < ForewordLen + 1 && f.tail == 0 {
		var buf bytes.Buffer
		buf.Write(f.tailbuf)
		buf.Write(data)
		f.tailbuf = buf.Bytes()
		return
	}

	// Copy the rest of data
	if len(f.tailbuf) > 0 {
		tmp := make([]byte, len(f.tailbuf) + len(data))
		copy(tmp[0:], f.tailbuf[0:])
		copy(tmp[len(f.tailbuf):], data[0:])
		data = tmp
		f.tailbuf = nil
	}

	if f.tail == 0 && f.probe(data) {
		// Calculate frame length and allocate data buffer
		r := bytes.NewReader(data[MagicWordLen:MagicWordLen+DataFrameLen])
		var fl uint64

		binary.Read(r, binary.BigEndian, &fl)
		f.buf = make([]byte, fl)
		f.framelen = int(fl)

		f.copy(data[ForewordLen:])
	} else if f.tail > 0 {
		f.copy(data)
	} else {
		f.buf = data
	}
}

func (f *dataFrame) GetFrame() []byte {
	out := make([]byte, len(f.buf))
	copy(out[0:], f.buf[0:])
	f.isFull = false
	f.tail = 0
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
	buf := make([]byte, MagicWordLen + lbuf.Len() + frame.Len())
	copy(buf[0:MagicWordLen], dataFrameMagicWord[:]) // data frame magic word
	copy(buf[MagicWordLen:], lbuf.Bytes()) // length
	copy(buf[MagicWordLen+lbuf.Len():], frame.Bytes()) // serialized data structure
	return buf, nil
}

func (f *dataFrame) probe(buf []byte) bool {
	if len(buf) < len(dataFrameMagicWord) { return false }

	for i := 0; i < len(dataFrameMagicWord); i++ {
		if buf[i] != dataFrameMagicWord[i] { return false }
	}

	return true
}
