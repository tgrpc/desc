package main

import (
	"flag"
	"fmt"

	"github.com/tgrpc/desc"
)

var (
	imp, method string
)

func init() {
	flag.StringVar(&method, "m", "", "-m helloworld.Greeter/SayHello")
	// flag.StringVar(&imp, "I", ".", "-I $GOPATH/src/github.com/tgrpc/desc/google.protobuf")
}

func main() {
	flag.Parse()

	imps := flag.Args()
	fmt.Printf("imports: %+v\n", imps)
	desc.SearchDescSrcByRawDescs(method, nil, imps...)
}
