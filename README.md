# Preface
It's quite often in network programming to send large amount of structured data over TCP networks that do not fit
into a single TCP packet, hence there is a strong need to re-assemble received packets into
a single data frame taking into account packet fragmentation. This Go library is meant to
facilitate that.

## API
```go
type DataFrame interface {
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
```

## How to use
```go
    //
    // Client
    //
    df := netdataframe.NewDataFrame()    
    req := &Request{}
    req.msg = "Hello"
    
    // Send your data over wire
    frame, err := df.ToFrame(p)
    if err != nil {
        panic(err)
    }

    var n int
    n, err = conn.Write(frame)
    ...

    //
    // Server
    //
    // Read network data in a loop
    n, err := conn.Read(readbuf)
    
    // Capture data frames
    df.Capture(readbuf[0:n])
    if df.IsFullFrame() {
        // Say hello
        req := &Request{}
        err := df.Decode(req)
        if err != nil { panic(err) }
        fmt.Printf(req.msg) // hello
    }
    
````
