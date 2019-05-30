package desc

import (
	"fmt"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/tgrpc/grpcurl"
)

func decodeFileDescSet(method string, rawDescs []string) (*descriptor.FileDescriptorSet, error) {
	fileDescriptorSet, err := decodeDescFromRawBytes("", rawDescs)
	if err != nil {
		return nil, err
	}

	serviceName, err := getServiceName(method)
	if err != nil {
		return nil, err
	}
	service, err := GetServiceDescriptor([]*descriptor.FileDescriptorSet{fileDescriptorSet}, serviceName)
	if err != nil {
		return nil, err
	}
	fileDescriptorSet, err = SortFileDescriptorSet(service.FileDescriptorSet, service.FileDescriptorProto)
	if err != nil {
		return nil, err
	}
	return fileDescriptorSet, nil
}

func searchDescSrc(fileDescriptorSet *descriptor.FileDescriptorSet) (grpcurl.DescriptorSource, error) {
	source, err := grpcurl.DescriptorSourceFromFileDescriptorSet(fileDescriptorSet)
	if err != nil {
		fmt.Println(err)
	}
	return source, err
}

func SearchDescSrcByRawDescs(method string, rawDescs []string) (grpcurl.DescriptorSource, error) {
	fileDescriptorSet, err := decodeFileDescSet(method, rawDescs)
	if err != nil {
		return nil, err
	}
	return searchDescSrc(fileDescriptorSet)
}
