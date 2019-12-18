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

raw descs will be saved in file helloworld.Greeter.desc .
helloworld.Greeter.desc can be used in tgrpc.toml like this:

```
log_level = "debug"
[service]
[service.Greeter]
address = "localhost:2080"
reuse_desc = true
proto_base_path = "$GOPATH/src/github.com/tgrpc/ngrpc"
include_imports = "helloworld/helloworld.proto"
keepalive = "100s"
raw_descs = ['H4sIAAAAAAAC/2ySz47aMBDGa5wQGKQKTRGyOERRDlVOVIWqvbeHXji5Uu+BGLAUYogNq32Tve3r7HEfYcW+yMp2+LNib/59M6P55ksAyrxajbe1MgphLcpS3am6LNJfEM2kNlzscAhttVxqYRhJSBbyhnAAYSk30rCWkz2kKwhmebVChKDKN8LNdLl74wg6c1mbdZHfuyHKz4zfoHMQtZaq0owmNOtNvowvfsb/fY2fm9JHAlGj4mdoyaKx15IFMoiaPreny09obRVCLxj1tuzbHnKQ04lmQULtIQ6cqk2tWZjQrMs9+N6fPzRrJzSj3INTl1aNEpoR7iHl0PEZ6i1+hdAmrRlxt/Wvb7N5cV/GGMAok5d/1L46JXulpAnAX2HchNh9FPJkDz1b/ifqg1wI/A6BNYHv0mw+7WhwK+otTiFqduDwuuGyeHTj/3f/6RiT52NMXo4xeXiNP83b7q+avgUAAP//cpNxkGMCAAA=']
```