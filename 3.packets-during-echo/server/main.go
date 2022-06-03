package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"io"
	"log"
	"path"
	"runtime"
)

const (
	listenAddress = "127.0.0.1:4243"
	certPemPath="../../cert/cert.pem"
	privKeyPath="../../cert/priv.key"
)
var(
	certFile, keyFile   string
)
func main(){
	// todo: learn x509 and key pair
	var err error
	certs := make([]tls.Certificate, 1)
	certs[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		panic(err)
	}
	tlsConfig:=&tls.Config{
		Certificates: certs,
		NextProtos: []string{"echo-quic-demo"},
	}

	fmt.Println("Quic server is running, it will exit after a stream is done")
	listener,err:=quic.ListenAddr(listenAddress,tlsConfig,nil)
	if err != nil {
		log.Fatalln(err)
	}
	conn,err:=listener.Accept(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Connection is established\n")
	stream,err:= conn.AcceptStream(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Create a new stream id: %d\n",stream.StreamID())
	// good point!
	_,err=io.Copy(loggingWriter{Writer:stream},stream)

	if err != nil {
		fmt.Printf("stream %d is closed, err:%s\n",stream.StreamID(),err)
	}
}



func init() {
	certFile, keyFile = func() (string, string) {
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			panic("Failed to get current frame")
		}

		certFolderPath := path.Dir(filename)
		return path.Join(certFolderPath, certPemPath), path.Join(certFolderPath,privKeyPath)
	}()
}


// loggingWriter is a good example that how to wrap a type
// good point!
type loggingWriter struct {
	writeType string
	io.Writer
}


func (w loggingWriter) Write(b []byte)  (int, error) {
	fmt.Printf("Server: Got '%s'\n", string(b))
	return w.Writer.Write(b)
}

