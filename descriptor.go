package desc

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/sirupsen/logrus"
	"github.com/tgrpc/grpcurl"
)

var (
	lock sync.Mutex
	log  *logrus.Entry
)

func init() {
	SetLog("debug")
}

func SetLog(logLevel string) {
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Error(err)
		lvl = logrus.DebugLevel
	}
	logger := logrus.New()
	logger.SetLevel(lvl)
	log = logrus.NewEntry(logger)
}

// protoPath 与 incImp 不能有重叠部分 incImp例子：服务目录/服务proto文件
func GenDescriptorSet(protoPath, descSetOut, incImp string) error {
	if protoPath[0] != "/"[0] {
		var prefix string
		if protoPath[0] == "$"[0] {
			spl := strings.SplitN(protoPath, "/", 2)
			prefix = os.Getenv(spl[0][1:])
			protoPath = spl[1]
		} else {
			var err error
			prefix, err = os.Getwd()
			if isErr(err) {
				return err
			}
		}
		protoPath = filepath.Join(prefix, protoPath)
	}
	incImp = getServiceProto(incImp)
	args := []string{fmt.Sprintf("--proto_path=%s", protoPath), fmt.Sprintf("--descriptor_set_out=%s", descSetOut), "--include_imports", fmt.Sprintf("%s", incImp)}

	lock.Lock()
	bs, err := exec.Command("protoc", args...).CombinedOutput()
	lock.Unlock()
	if len(bs) > 0 {
		log.WithField("protoc", string(bs)).Error(err)
	}
	return err
}

func GetDescriptorSource(protoBasePath, method, incImp string, reuseDesc bool, rawDescs []string) (grpcurl.DescriptorSource, error) {
	fileDescriptorSet, err := GetDescriptor(protoBasePath, method, incImp, reuseDesc, rawDescs)
	if isErr(err) {
		return nil, err
	}

	fileDescriptorSet, err = DecodeFileDescriptorSet(method, fileDescriptorSet)
	if isErr(err) {
		return nil, err
	}

	return grpcurl.DescriptorSourceFromFileDescriptorSet(fileDescriptorSet)
}

// method: pkg.Service incImp:pkg.service.proto
func GetDescriptor(protoBasePath, method, incImp string, reuseDesc bool, rawDescs []string) (*descriptor.FileDescriptorSet, error) {
	serviceName, err := GetServiceName(method)
	if isErr(err) {
		return nil, err
	}
	descSetOut := "." + serviceName + ".pbin"

	var desc *descriptor.FileDescriptorSet

	if len(rawDescs) > 0 {
		desc, err := DecodeFileDescriptorSetByRaw(descSetOut, rawDescs)
		if err == nil {
			return desc, nil
		}
		log.Errorf("%+v", err)
	}

	if reuseDesc {
		desc, err := decodeDesc(descSetOut)
		if err == nil {
			log.WithField("FileDescriptorSet", descSetOut).Debug("use exist desc")
			return desc, nil
		}
		log.Errorf("%+v", err)
	}

	err = GenDescriptorSet(protoBasePath, descSetOut, incImp)
	if isErr(err) {
		return nil, err
	}
	desc, err = decodeDesc(descSetOut)
	if isErr(err) {
		return nil, err
	}
	return desc, nil
}

func getServiceProto(protoFile string) string {
	spl := strings.Split(protoFile, "/")
	size := len(spl)
	if size <= 2 {
		return protoFile
	}
	return strings.Join(spl[size-2:], "/")
}

// helloworld.Greeter
func GetServiceName(method string) (string, error) {
	spl := strings.Split(method, "/")
	size := len(spl)
	if size < 2 {
		return "", fmt.Errorf("invalid gRPC method: %s", method)
	}
	return spl[size-2], nil
}

// exp: helloworld.Greeter/SayHello
func GetMethod(method string) (string, error) {
	split := strings.Split(method, "/")
	if len(split) < 2 {
		return "", fmt.Errorf("invalid gRPC method: %s", method)
	}
	return strings.Join(split[len(split)-2:], "/"), nil
}

func decodeDesc(descriptorSetFilePath string) (*descriptor.FileDescriptorSet, error) {
	log.Infof("decode desc...")
	data, err := ioutil.ReadFile(descriptorSetFilePath)
	if err != nil {
		return nil, err
	}
	fileDescriptorSet := &descriptor.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fileDescriptorSet); err != nil {
		return nil, err
	}
	return fileDescriptorSet, nil
}

// Descriptor is an extracted service.
type ServiceDescriptor struct {
	*descriptor.ServiceDescriptorProto

	FullyQualifiedPath  string
	FileDescriptorProto *descriptor.FileDescriptorProto
	FileDescriptorSet   *descriptor.FileDescriptorSet
}

func GetServiceDescriptor(fileDescriptorSets []*descriptor.FileDescriptorSet, path string) (*ServiceDescriptor, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}
	if path[0] == '.' {
		path = path[1:]
	}

	var serviceDescriptorProto *descriptor.ServiceDescriptorProto
	var fileDescriptorProto *descriptor.FileDescriptorProto
	var fileDescriptorSet *descriptor.FileDescriptorSet
	for _, iFileDescriptorSet := range fileDescriptorSets {
		for _, iFileDescriptorProto := range iFileDescriptorSet.File {
			iServiceDescriptorProto, err := findServiceDescriptorProto(path, iFileDescriptorProto)
			if err != nil {
				return nil, err
			}
			if iServiceDescriptorProto != nil {
				if serviceDescriptorProto != nil {
					return nil, fmt.Errorf("duplicate services for path %s", path)
				}
				serviceDescriptorProto = iServiceDescriptorProto
				fileDescriptorProto = iFileDescriptorProto
			}
		}
		// return first fileDescriptorSet that matches
		// as opposed to duplicate check within fileDescriptorSet, we easily could
		// have multiple fileDescriptorSets that match
		if serviceDescriptorProto != nil {
			fileDescriptorSet = iFileDescriptorSet
			break
		}
	}
	if serviceDescriptorProto == nil {
		return nil, fmt.Errorf("no service for path %s", path)
	}
	return &ServiceDescriptor{
		ServiceDescriptorProto: serviceDescriptorProto,
		FullyQualifiedPath:     "." + path,
		FileDescriptorProto:    fileDescriptorProto,
		FileDescriptorSet:      fileDescriptorSet,
	}, nil
}

func decodeDescByRaw(raw string) (*descriptor.FileDescriptorProto, error) {
	data := ParseStr2Bytes(raw)
	return ExtractFile(data)
}

func DecodeFileDescriptorSetByRaw(descSetOut string, raws []string) (*descriptor.FileDescriptorSet, error) {
	log.Infof("decode desc frow raw...")
	descSet := new(descriptor.FileDescriptorSet)
	descSet.File = make([]*descriptor.FileDescriptorProto, 0, len(raws))
	for _, raw := range raws {
		descProto, err := decodeDescByRaw(raw)
		if err != nil {
			return nil, err
		}
		descSet.File = append(descSet.File, descProto)
	}
	return descSet, nil
}

func DecodeFileDescriptorSet(method string, fileDescriptorSet *descriptor.FileDescriptorSet) (*descriptor.FileDescriptorSet, error) {
	serviceName, err := GetServiceName(method)
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

func SortFileDescriptorSet(fileDescriptorSet *descriptor.FileDescriptorSet, fileDescriptorProto *descriptor.FileDescriptorProto) (*descriptor.FileDescriptorSet, error) {
	// best-effort checks
	names := make(map[string]struct{}, len(fileDescriptorSet.File))
	for _, iFileDescriptorProto := range fileDescriptorSet.File {
		if iFileDescriptorProto.GetName() == "" {
			return nil, fmt.Errorf("no name on FileDescriptorProto")
		}
		if _, ok := names[iFileDescriptorProto.GetName()]; ok {
			return nil, fmt.Errorf("duplicate FileDescriptorProto in FileDescriptorSet: %s", iFileDescriptorProto.GetName())
		}
		names[iFileDescriptorProto.GetName()] = struct{}{}
	}
	if _, ok := names[fileDescriptorProto.GetName()]; !ok {
		return nil, fmt.Errorf("no FileDescriptorProto named %s in FileDescriptorSet with names %v", fileDescriptorProto.GetName(), names)
	}
	newFileDescriptorSet := &descriptor.FileDescriptorSet{}
	for _, iFileDescriptorProto := range fileDescriptorSet.File {
		if iFileDescriptorProto.GetName() != fileDescriptorProto.GetName() {
			newFileDescriptorSet.File = append(newFileDescriptorSet.File, iFileDescriptorProto)
		}
	}
	newFileDescriptorSet.File = append(newFileDescriptorSet.File, fileDescriptorProto)
	return newFileDescriptorSet, nil
}

// TODO: we don't actually do full path resolution per the descriptor.proto spec
// https://github.com/google/protobuf/blob/master/src/google/protobuf/descriptor.proto#L185
func findDescriptorProto(path string, fileDescriptorProto *descriptor.FileDescriptorProto) (*descriptor.DescriptorProto, error) {
	if fileDescriptorProto.GetPackage() == "" {
		return nil, fmt.Errorf("no package on FileDescriptorProto")
	}
	if !strings.HasPrefix(path, fileDescriptorProto.GetPackage()) {
		return nil, nil
	}
	return findDescriptorProtoInSlice(path, fileDescriptorProto.GetPackage(), fileDescriptorProto.GetMessageType())
}

func findDescriptorProtoInSlice(path string, nestedName string, descriptorProtos []*descriptor.DescriptorProto) (*descriptor.DescriptorProto, error) {
	var foundDescriptorProto *descriptor.DescriptorProto
	for _, descriptorProto := range descriptorProtos {
		if descriptorProto.GetName() == "" {
			return nil, fmt.Errorf("no name on DescriptorProto")
		}
		fullName := nestedName + "." + descriptorProto.GetName()
		if path == fullName {
			if foundDescriptorProto != nil {
				return nil, fmt.Errorf("duplicate messages for path %s", path)
			}
			foundDescriptorProto = descriptorProto
		}
		nestedFoundDescriptorProto, err := findDescriptorProtoInSlice(path, fullName, descriptorProto.GetNestedType())
		if err != nil {
			return nil, err
		}
		if nestedFoundDescriptorProto != nil {
			if foundDescriptorProto != nil {
				return nil, fmt.Errorf("duplicate messages for path %s", path)
			}
			foundDescriptorProto = nestedFoundDescriptorProto
		}
	}
	return foundDescriptorProto, nil
}

func findServiceDescriptorProto(path string, fileDescriptorProto *descriptor.FileDescriptorProto) (*descriptor.ServiceDescriptorProto, error) {
	if fileDescriptorProto.GetPackage() == "" {
		return nil, fmt.Errorf("no package on FileDescriptorProto")
	}
	if !strings.HasPrefix(path, fileDescriptorProto.GetPackage()) {
		return nil, nil
	}
	var foundServiceDescriptorProto *descriptor.ServiceDescriptorProto
	for _, serviceDescriptorProto := range fileDescriptorProto.GetService() {
		if fileDescriptorProto.GetPackage()+"."+serviceDescriptorProto.GetName() == path {
			if foundServiceDescriptorProto != nil {
				return nil, fmt.Errorf("duplicate services for path %s", path)
			}
			foundServiceDescriptorProto = serviceDescriptorProto
		}
	}
	return foundServiceDescriptorProto, nil
}

func isErr(err error) bool {
	return err != nil
}
