package netdataframe_test

import (
	"bytes"
	"time"
	netdataframe "github.com/AgentCoop/net-dataframe"
	"math/rand"
	"testing"
)

const (
	MinBulkDataLen = 1500
	MaxBulkDataLen = 65535
	MinPacketDataLen = 1
	MaxPacketDataLen = 1500 // (MTU for Ethernet)
)

type Payload struct {
	A int8
	BulkData []byte
	B int16
	C uint64
	Msg string
}

var (
	df = netdataframe.NewDataFrame()
	randomStr1 string = "Hello, World!"
)

func randFromRange(min int, max  int) int {
	if max <= min {
		return min
	} else {
		return min + int(rand.Int31n(int32(max) - int32(min)) + 1)
	}
}

func generatePayload() *Payload {
	a := int8(randFromRange(0, 127))
	b := int16(randFromRange(0, 32767))
	c := rand.Uint64()
	msg := randomStr1

	datalen := randFromRange(MinBulkDataLen, MaxBulkDataLen)
	bulkData := make([]byte, datalen)
	_, err := rand.Read(bulkData)
	if err != nil { panic(err) }

	return &Payload{a, bulkData, b, c,msg}
}

func transferPayload(input ...*Payload) []*Payload {
	var b bytes.Buffer
	output := make([]*Payload, len(input))

	for _, p := range input {
		df, err := df.ToFrame(p)
		if err != nil { panic(err) }
		b.Write(df)
	}

	// Split data in chunks of variable length to simulate network data transmission
	data := b.Bytes()
	pivots := make([]int, 1)
	pivots[0] = 0
	for ; pivots[len(pivots) - 1] < len(data); {
		plen := randFromRange(MinPacketDataLen, MaxPacketDataLen)
		l := pivots[len(pivots)-1] + 1
		r := l + plen
		if r > len(data) {
			r = len(data)
		}
		pivots = append(pivots, randFromRange(l, r))
	}

	var j int
	for i := 0; i < len(pivots) - 1; i++ {
		chunk := data[pivots[i]:pivots[i+1]]
		df.Capture(chunk)
		if df.IsFullFrame() {
			recvPayload := &Payload{}
			err := df.Decode(recvPayload)
			if err != nil { panic(err) }
			output[j] = recvPayload
			j++
		}
	}

	return output
}

var counter int

func testPayload(t *testing.T, sendp *Payload, recvp *Payload) {
	if sendp == nil || recvp == nil {
		t.Fatalf("nil pointer")
	}
	if sendp.A != recvp.A {
		t.Fatalf("expected A %d, got %d\n", sendp.A, recvp.A)
	}
	if sendp.B != recvp.B {
		t.Fatalf("expected B %d, got %d\n", sendp.B, recvp.B)
	}
	if sendp.C != recvp.C {
		t.Fatalf("expected C %d, got %d\n", sendp.C, recvp.C)
	}
	if sendp.Msg != recvp.Msg {
		t.Fatalf("expected string %s, got %s\n", sendp.Msg, recvp.Msg)
	}
	for i := 0; i < len(sendp.BulkData); i++ {
		if sendp.BulkData[i] != recvp.BulkData[i] {
			t.Fatalf("bulk data expected byte[%d] %x, got %x\n", i, sendp.BulkData[i], recvp.BulkData[i])
		}
	}
	if sendp.Msg != recvp.Msg {
		t.Fatalf("expected A %d, got %d\n", sendp.A, recvp.A)
	}
}

func testSuite(t *testing.T, i int) {
	//defer func() {
	//	if r := recover(); r != nil {
	//		breakpoint := 0
	//		breakpoint++
	//	}
	//}()

	p1 := generatePayload()
	p2 := generatePayload()
	p3 := generatePayload()
	p4 := generatePayload()
	r := transferPayload(p1, p2, p3, p4)

	testPayload(t, p1, r[0])
	testPayload(t, p2, r[1])
	testPayload(t, p3, r[2])
	testPayload(t, p4, r[3])
}

func Test_DataFrame(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	for counter = 0; counter < 5000; counter++ {
		testSuite(t, counter)
	}
}
