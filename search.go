package desc

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/tgrpc/grpcurl"
	"github.com/toukii/goutils"
)

type ProtoDesc struct {
	fileDescriptorSet *descriptor.FileDescriptorSet
}

func NewProtoDesc(fileDescriptorSet *descriptor.FileDescriptorSet, pbDirs ...string) *ProtoDesc {
	pdesc := &ProtoDesc{
		fileDescriptorSet: fileDescriptorSet,
	}

	for _, dir := range pbDirs {
		Collect(dir)
	}

	return pdesc
}

func (p *ProtoDesc) searchDescSrc() (grpcurl.DescriptorSource, error) {
	var source grpcurl.DescriptorSource
	for {
		var err error
		source, err = grpcurl.DescriptorSourceFromFileDescriptorSet(p.fileDescriptorSet)
		if err == nil {
			break
		}

		msg := err.Error()
		protoFile := needProtoFile(msg)
		pbgoFile := parse2PBFile(protoFile)
		log.Debugf("%s ==> %s", msg, protoFile)
		pbgo := Search(pbgoFile)
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
		p.fileDescriptorSet.File = append(p.fileDescriptorSet.File, pbdescfile)
	}
	return source, nil
}

func SearchDescSrcByRawDescs(method string, rawDescs []string, pbDirs ...string) (grpcurl.DescriptorSource, error) {
	fileDescriptorSet, err := DecodeFileDescriptorSetByRaw("", rawDescs)
	if err != nil {
		return nil, err
	}
	fileDescriptorSet, err = DecodeFileDescriptorSet(method, fileDescriptorSet)
	if err != nil {
		return nil, err
	}
	pdesc := NewProtoDesc(fileDescriptorSet, pbDirs...)

	return pdesc.searchDescSrc()
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
	pa := regexp.MustCompile(fmt.Sprintf(`(?s)bytes of a gzipped FileDescriptorProto\n((.*)),\n}`))
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
