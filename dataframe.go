package netdataframe

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
)

var (
	// head -c 8 /dev/urandom | hexdump -C
	dataFrameMagicWord = [...]byte{ 0xaa, 0xde, 0x07, 0xa6, 0x02, 0x57, 0xc2, 0xd0 }

	ErrIncompleteFrame = errors.New("netdataframe.Decode: trying to decode incomplete frame")

	ErrFrameFull = errors.New("netdataframe.capture: frame is full")

	ErrRawData = errors.New("netdataframe.capture: trying to capture raw data stream")
)

const (
	MagicWordLen = len(dataFrameMagicWord)
	DataFrameLen = 8
	ForewordLen = MagicWordLen + DataFrameLen
)

type dataFrame struct {
	data      []byte
}

type receiver struct {
	buf      []byte
	head     int
	tailbuf  []byte
	isFull   bool
	framelen int
}

func (r *receiver) copy(data []byte) {
	n := copy(r.buf[r.head:], data[0:])
	r.head += n
	if r.head == r.framelen {
		rest := len(data) - n
		if rest > 0 {
			r.tailbuf = make([]byte, rest)
			copy(r.tailbuf[0:], data[n:])
		}
		r.isFull = true
	}
}

func (recv *receiver) capture(data []byte) error {
	if recv.isFull {
		return ErrFrameFull
	}
	// capture foreword plus one extra byte to move frame head
	if len(recv.tailbuf) + len(data) < ForewordLen + 1 && recv.head == 0 {
		var buf bytes.Buffer
		buf.Write(recv.tailbuf)
		buf.Write(data)
		recv.tailbuf = buf.Bytes()
		return nil
	}
	// Flush tail buffer
	if len(recv.tailbuf) > 0 {
		tmp := make([]byte, len(recv.tailbuf) + len(data))
		copy(tmp[0:], recv.tailbuf[0:])
		copy(tmp[len(recv.tailbuf):], data[0:])
		data = tmp
		recv.tailbuf = nil
	}
	//
	if recv.head == 0 && recv.probe(data) {
		// Calculate frame length and allocate data buffer
		r := bytes.NewReader(data[MagicWordLen:MagicWordLen+DataFrameLen])
		var fl uint64

		binary.Read(r, binary.BigEndian, &fl)
		recv.buf = make([]byte, fl)
		recv.framelen = int(fl)

		recv.copy(data[ForewordLen:])
	} else if recv.head > 0 {
		recv.copy(data)
	} else {
		return ErrRawData
	}
	return nil
}

func (r *receiver) getFrame() *dataFrame {
	frame := &dataFrame{}
	frame.data = make([]byte, len(r.buf))
	copy(frame.data[0:], r.buf[0:])
	r.isFull = false
	r.head = 0
	return frame
}

func (r *receiver) Capture(data []byte) ([]*dataFrame, error) {
	frames := make([]*dataFrame, 0)
	err := r.capture(data)
	if err != nil { return nil, err }
	empty := []byte{}
	for r.isFull {
		frames = append(frames, r.getFrame())
		err := r.capture(empty) // read from the tail buffer
		if err != nil { panic(err) }
	}
	return frames, nil
}

func (f *receiver) Flush() []byte {
	out := make([]byte, len(f.buf))
	copy(out[0:], f.buf[0:])
	f.head = 0
	f.tailbuf = nil
	f.isFull = false
	return out
}

func ToFrame(data interface{}) (*dataFrame, error) {
	var buf bytes.Buffer
	// Encode frame data
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil { return nil, err }

	// Write frame length in big endian order
	framelen := uint64(buf.Len())
	lbuf := new(bytes.Buffer)
	err = binary.Write(lbuf, binary.BigEndian, framelen)
	if err != nil { return nil, err }

	// Compose data stream
	frame := &dataFrame{}
	frame.data = make([]byte, MagicWordLen + lbuf.Len() + buf.Len())
	copy(frame.data[0:MagicWordLen], dataFrameMagicWord[:]) // data frame magic word
	copy(frame.data[MagicWordLen:], lbuf.Bytes()) // length
	copy(frame.data[MagicWordLen+lbuf.Len():], buf.Bytes()) // serialized data structure
	return frame, nil
}

func (f *receiver) probe(buf []byte) bool {
	if len(buf) < len(dataFrameMagicWord) { return false }

	for i := 0; i < len(dataFrameMagicWord); i++ {
		if buf[i] != dataFrameMagicWord[i] { return false }
	}

	return true
}

func (f *dataFrame) GetBytes() []byte {
	return f.data
}

func (f *dataFrame) Decode(receiver interface{}) error {
	buf := bytes.NewBuffer(f.data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(receiver)
	return err
}
