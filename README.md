# desc
search descriptors from go source code for tgrpc


### google.protobuf/descriptor

```
protoc --gogo_out=plugins=grpc:. google.protobuf/descriptor.proto
```


```
go get github.com/tgrpc/desc/desc

desc -m helloworld.Greeter/SayHello '$GOPATH/src/github.com/tgrpc'
```