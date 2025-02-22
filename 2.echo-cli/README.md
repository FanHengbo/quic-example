# Echo cli
The echo-cli provides a more controllable way for you to try it. You will learn some quic behaviors during 
try it. I will also discuss some details found in programming.
Running `go run main.go` both in server and client could get a echo demo of quic.

## The key pair and certification
In the echo server the `tls.Config` is generated automatically. In this topic, we will take an eye on it.
There are public-private key pair and certification when creating use the TLS. It uses asymmetric 
cryptography to encrypt the key used in the both side to transport data with a CA certificate.  
1. Server passes the public key to client. A CA is used to certificate the public key to avoid revising during transport.
2. Client generates a key to encrypt transport data, sends to server after encrypting by the public key.
3. Server decrypts the encrypted key and get the key generated by client.

## A good wrapper with io.copy
The server side implements a `loggingWriter` to wrap `io.writer` and print some information. For example, if we want to
print some information before writing, we can do this:
```go
var stream quic.Stream
data:="Data I want to write"
fmt.Printf("Write data: %s to stream\n",data)
stream.Write([]byte(data))
```
The code up could achieve the printing purpose. However, it's not good as:
- the print logic is independent with the writing.  
The printing should connect with the writing logic, so it's easy to manage the printing logic.
  
- hard to manage code when the project is large  
Every writing action should bind a printing logic, but it's easy to miss.
  
As a result, wrap the writer and print in the wrapper is a better solution:
```go
type loggingWriter struct {
	io.Writer
}

func (w loggingWriter) Write(b []byte)  (int, error) {
	fmt.Printf("Server: Got '%s'\n", string(b))
	return w.Writer.Write(b)
}
```

## What if the server send the full echo message with two writing operation?
Use the wrapper, we can easily integrate our new logical into the wrapper like this:
```go
func (w loggingWriter) Write(b []byte)  (int, error) {
	fmt.Printf("Server: Got '%s'\n", string(b))
	if w.writeType=="twice"{
		l:=len(b)
		n,err:=w.Writer.Write(b[:l/2])
		nn,err:=w.Writer.Write(b[l/2:])
		return n+nn,err
	}
	return w.Writer.Write(b)
}
```
Based on the testing, we can find that it doesn't affect the client side receiving.

## Difference between OpenStream and OpenStreamSync?
See the document:  
- OpenStream.  
  OpenStream opens a new bidirectional QUIC stream. 
  There is no signaling to the peer about new streams: 
  The peer can only accept the stream after data has been sent on the stream. 
  If the error is non-nil, it satisfies the net.Error interface. 
  When reaching the peer's stream limit, err.Temporary() will be true. 
  If the connection was closed due to a timeout, Timeout() will be true.
- OpenStreamSync.   
  OpenStreamSync opens a new bidirectional QUIC stream. 
  It blocks until a new stream can be opened. 
  If the error is non-nil, it satisfies the net.Error interface. 
  If the connection was closed due to a timeout, Timeout() will be true.

The difference is whether the peer can know and accept the new stream. For echo case here, there is no more different.

## What happened after the server side finish write data in a stream?

Based on the RFC 9000, the Bidirectional stream states flow figure is:
```go
o
| Create Stream (Sending)
| Peer Creates Bidirectional Stream
v
+-------+
| Ready | Send RESET_STREAM
|       |-----------------------.
+-------+                       |
|                           |
| Send STREAM /             |
|      STREAM_DATA_BLOCKED  |
v                           |
+-------+                       |
| Send  | Send RESET_STREAM     |
|       |---------------------->|
+-------+                       |
|                           |
| Send STREAM + FIN         |
v                           v
+-------+                   +-------+
| Data  | Send RESET_STREAM | Reset |
| Sent  |------------------>| Sent  |
+-------+                   +-------+
|                           |
| Recv All ACKs             | Recv ACK
v                           v
+-------+                   +-------+
| Data  |                   | Reset |
| Recvd |                   | Recvd |
+-------+                   +-------+
```
The RFC 9000 tells:
> In the "Send" state, an endpoint transmits -- and retransmits as necessary -- stream data in
STREAM frames. The endpoint respects the ﬂow control limits set by its peer and continues to
accept and process MAX_STREAM_DATA frames. An endpoint in the "Send" state generates
STREAM_DATA_BLOCKED frames if it is blocked from sending by stream ﬂow control limits.
> After the application indicates that all stream data has been sent and a STREAM frame
containing the FIN bit is sent, the sending part of the stream enters the "Data Sent" state.

So in the echo case, after writing the stream, the server stream still keep the `Send` state as the application doesn't
indicate "all stream data has been sent".
So the server stream have to wait a timeout(io.Copy waits the data from a stream until an EOF) and then the stream 
will move to the `Sent` state(not sure, the timeout might to the Data Recvd directly).

## Why the server could only serve one client request?
The first client request gets an echo message, but the second one blocks until gets a "timeout: 
no recent network activity" error.  
So why the server doesn't exit but cannot serve a new client request?

The reason is our server only set logic for the first stream. If the client runs twice, the second one tries to create 
a new stream. However, there is no logic in server side to solve it and the request fails at the echo example here.
