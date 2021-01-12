# Preface
It's quite often in network programming to send large amount of structured data over TCP networks that do not fit
into a single TCP packet, hence there is a strong need to re-assemble received packets into
a single data frame taking into account packet fragmentation. This Go library is meant to
facilitate that.

### API
```go
type Receiver interface {
	// Captures all available frames in buffer
	Capture(buf []byte) ([]*dataFrame, error)
	// Resets frame capturing returning captured before data
	Flush() []byte
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
```

### How to use
```go
    //
    // Client
    //
    req := &Request{}
    req.msg = "Hello"
    
    // Send your data over wire
    frame, err := netdataframe.ToFrame(p)
    if err != nil {
        panic(err)
    }

    var n int
    n, err = conn.Write(frame.GetBytes())
    ...

    //
    // Server
    //
    // Read network data
    n, err := conn.Read(readbuf)

    // Capture data frames
    recv := netdataframe.NewReceiver()
    frames, err := recv.Capture(readbuf[0:n])
    // Say hello
    req := &Request{}
    err := frames[0].Decode(req)
    if err != nil { panic(err) }
    fmt.Printf(req.msg) // hello    
````
