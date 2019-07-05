package desc

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/tgrpc/grpcurl"
	"github.com/toukii/bytes"
	"github.com/toukii/goutils"
)

type ProtoDesc struct {
	fileDescriptorSet *descriptor.FileDescriptorSet
	rawPbDescs        []string
}

func NewProtoDesc(fileDescriptorSet *descriptor.FileDescriptorSet, rawDescs []string, pbDirs ...string) *ProtoDesc {
	pdesc := &ProtoDesc{
		fileDescriptorSet: fileDescriptorSet,
		rawPbDescs:        make([]string, 0, 3+len(rawDescs)),
	}
	pdesc.rawPbDescs = append(pdesc.rawPbDescs, rawDescs...)

	Collect(pbDirs...)
	for key, it := range associPB {
		fmt.Println(key, it)
	}

	return pdesc
}

func (p *ProtoDesc) searchDescSrc(method string) (grpcurl.DescriptorSource, error) {
	var source grpcurl.DescriptorSource
	for {
		var err error
		source, err = grpcurl.DescriptorSourceFromFileDescriptorSet(p.fileDescriptorSet)
		if err == nil && len(p.fileDescriptorSet.GetFile()) > 0 {
			log.Errorf("DescriptorSourceFromFileDescriptorSet, err:%+v", err)
			break
		}

		var msg, protoFile, pbgoFile string
		var pbgo *AssociPB
		if err == nil {
			msg = fmt.Sprintf("no descriptor found for %s", method)
			pkgName, _ := GetPackageName(method)
			serviceName, _ := GetServiceName(method)
			pbgo = SearchByPkgName(pkgName, serviceName)
		} else {
			msg = err.Error()
			protoFile = needProtoFile(msg)
			pbgoFile = parse2PBFile(protoFile)
			log.Debugf("%s ==> %s", msg, protoFile)
			pbgo = Search(pbgoFile)
		}
		if pbgo == nil {
			panic(fmt.Sprintf("cannot find %s", pbgoFile))
		}
		if len(pbgo.PBs) > 1 {
			pmsg := fmt.Sprintf("ambiguous %s files: ", pbgo.PBs[0].ImportName())
			for _, it := range pbgo.PBs {
				pmsg = pmsg + "\n " + it.AbsDir
			}
			panic(pmsg)
		}
		log.Debugf("search %s ==> %s", protoFile, pbgo.PBs[0].AbsDir)
		pbdesc := getDescriptorBytes(goutils.ReadFile(pbgo.PBs[0].AbsDir))
		pbdescfile, err := decodeDescByRaw(pbdesc)
		if err != nil {
			return source, err
		}
		p.rawPbDescs = append(p.rawPbDescs, pbdesc)
		p.fileDescriptorSet.File = append(p.fileDescriptorSet.File, pbdescfile)
	}
	return source, nil
}

func SearchDescSrcByRawDescs(method string, rawDescs []string, pbDirs ...string) (grpcurl.DescriptorSource, error) {
	var fileDescriptorSet *descriptor.FileDescriptorSet
	var err error
	if len(rawDescs) > 0 {
		fileDescriptorSet, err = DecodeFileDescriptorSetByRaw("", rawDescs)
		if err != nil {
			log.Errorf("DecodeFileDescriptorSetByRaw, err:%+v", err)
		}
		fileDescriptorSet, err = DecodeFileDescriptorSet(method, fileDescriptorSet)
		if err != nil {
			log.Errorf("DecodeFileDescriptorSet, err:%+v", err)
		}
	}

	if fileDescriptorSet == nil {
		fileDescriptorSet = &descriptor.FileDescriptorSet{
			File: make([]*descriptor.FileDescriptorProto, 0, 3),
		}
	}

	pdesc := NewProtoDesc(fileDescriptorSet, rawDescs, pbDirs...)

	descsrc, err := pdesc.searchDescSrc(method)
	serviceName, _ := GetServiceName(method)
	wr := bytes.NewWriter(make([]byte, 0, 10240))
	for i, it := range pdesc.rawPbDescs {
		if i > 0 {
			wr.Write([]byte(`, `))
		}
		wr.Write([]byte(`"`))
		wr.Write(goutils.ToByte(it))
		wr.Write([]byte(`"`))
	}
	goutils.WriteFile(fmt.Sprintf("%s.desc", serviceName), wr.Bytes())
	return descsrc, err
}

func Try(method string, pbDirs ...string) (grpcurl.DescriptorSource, error) {
	Collect(pbDirs...)
	serviceName, err := GetServiceName(method)
	if err != nil {
		return nil, err
	}
	importName := strings.ToLower(strings.Replace(serviceName, ".", "/", -1))
	apb := SearchByImportName(importName)
	if len(apb.PBs) <= 0 {
		return nil, fmt.Errorf("pkg not found")
	}
	fmt.Println("SearchByImportName:", apb)
	for _, it := range apb.PBs {
		pbdesc := getDescriptorBytes(goutils.ReadFile(it.AbsDir))
		// fmt.Println("pbdesc:", pbdesc)
		fileDescriptor, err := decodeDescByRaw(pbdesc)
		if err != nil {
			log.Errorf("%s err:%+v", it.ImportName(), err)
			continue
		}
		set := &descriptor.FileDescriptorSet{
			File: []*descriptor.FileDescriptorProto{fileDescriptor},
		}
		pb := NewProtoDesc(set, []string{pbdesc}, pbDirs...)
		source, err := pb.searchDescSrc(method)
		if err != nil {
			log.Errorf("searchDescSrc %s err:%+v", it.ImportName(), err)
			continue
		}
		return source, nil
	}
	return nil, fmt.Errorf("not found!")
}

func parse2PBFile(protofile string) string {
	return fmt.Sprintf("%s.pb.go", protofile[:len(protofile)-6])
}

func needProtoFile(err string) string {
	pa := regexp.MustCompile(fmt.Sprintf(`^no descriptor found for "([\S]+)"$`))
	ma := pa.FindStringSubmatch(err)
	size := len(ma)
	if size > 0 {
		return ma[size-1]
	}
	return ""
}

func getDescriptorBytes(bs []byte) string {
	cotxt := goutils.ToString(bs)
	pa := regexp.MustCompile(fmt.Sprintf(`(?s)bytes of a gzipped FileDescriptorProto\n((.*?)),\n}`))
	ma := pa.FindStringSubmatch(cotxt)
	size := len(ma)
	if size > 0 {
		return trim(ma[size-1])
	}
	return ""
}

func trim(ctx string) string {
	return strings.Trim(strings.Replace(ctx, "\n", "", -1), " ")
}
