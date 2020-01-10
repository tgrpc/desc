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
	unique            map[string]bool
}

func NewProtoDesc(fileDescriptorSet *descriptor.FileDescriptorSet, rawDescs []string, pbDirs ...string) *ProtoDesc {
	pdesc := &ProtoDesc{
		fileDescriptorSet: fileDescriptorSet,
		rawPbDescs:        make([]string, 0, 3+len(rawDescs)),
		unique:            make(map[string]bool, 10),
	}
	pdesc.rawPbDescs = append(pdesc.rawPbDescs, rawDescs...)

	Collect(pbDirs...)
	log.Infof("collect %d pb.go", len(associPB))
	for key, _ := range associPB {
		fmt.Println(key)
	}

	return pdesc
}

func (p *ProtoDesc) addDesc(unique, pbdesc string, pbdescfile *descriptor.FileDescriptorProto) {
	if p.unique[unique] {
		log.Warnf("%s is duplicated", unique)
		return
	}
	p.rawPbDescs = append(p.rawPbDescs, pbdesc)
	p.fileDescriptorSet.File = append(p.fileDescriptorSet.File, pbdescfile)
	p.unique[unique] = true
}

func (p *ProtoDesc) searchDescSrc(method string) (grpcurl.DescriptorSource, error) {
	var source grpcurl.DescriptorSource

	pkgName, _ := GetPackageName(method)
	serviceName, _ := GetServiceName(method)
	log.Debugf("search %s, pkg: %s, service: %s", method, pkgName, serviceName)

	for {
		var err error
		source, err = grpcurl.DescriptorSourceFromFileDescriptorSet(p.fileDescriptorSet)
		if err == nil && len(p.fileDescriptorSet.GetFile()) > 0 {
			log.Warnf("search desc loop end")
			break
		}

		var msg, protoFile, pbgoFile string
		var pbgo *AssociPB
		if err == nil {
			msg = fmt.Sprintf("no descriptor found for %s", method)
			pbgo = SearchByPkgName(pkgName, serviceName)
		} else {
			msg = err.Error()
			protoFile = needProtoFile(msg)
			pbgoFile = pkgName + "/" + parse2PBFile(protoFile)
			log.Debugf("%s ==> %s %s", msg, protoFile, pbgoFile)
			pbgo = Search(pbgoFile)
		}
		if pbgo == nil {
			panic(fmt.Sprintf("cannot find %s", pbgoFile))
		}

		var pbdesc string
		find := false
		var pbdescfile *descriptor.FileDescriptorProto
		for _, pb := range pbgo.PBs {
			log.Debugf("searching %s", pb.ImportName())
			pbfilebs := goutils.ReadFile(pb.AbsDir)
			pbfilecnt := goutils.ToString(pbfilebs)

			methodContains := strings.Contains(pbfilecnt, method)
			if !methodContains && strings.Contains(pbfilecnt, "Methods: []grpc.MethodDesc") {
				continue
			}

			log.Infof("%s --> %s", method, pb.AbsDir)

			pbdesc = getDescriptorBytes(pbfilebs)
			var err error
			pbdescfile, err = decodeDescByRaw(pbdesc)
			if err != nil {
				log.Errorf("err:%+v", err)
				continue
			} else {
				find = true
				p.addDesc(pb.ImportName(), pbdesc, pbdescfile)
			}
		}
		if !find {
			panic(fmt.Sprintf("not found pb for:%s", method))
		}
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
		} else {
			fileDescriptorSet, err = DecodeFileDescriptorSet(method, fileDescriptorSet)
			if err != nil {
				log.Errorf("DecodeFileDescriptorSet, err:%+v", err)
			}
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
		} else {
			wr.Write([]byte(`[`))
		}
		base64bs := []byte(Base64Encode(ParseStr2Bytes(it)))
		wr.Write([]byte(`'`))
		wr.Write(base64bs)
		wr.Write([]byte(`'`))
	}
	wr.Write([]byte(`]`))

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
