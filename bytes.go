package desc

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"

	gogoproto "github.com/gogo/protobuf/proto"
	gogodescriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

func Base64Encode(bs []byte) string {
	return base64.StdEncoding.EncodeToString(bs)
}

func Base64Decode(str string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(str)
}

func ParseStr2Bytes(str string) []byte {
	s := strings.Split(str, ",")
	return Parse2Bytes(s)
}

func Parse2Bytes(strs []string) []byte {
	bs := make([]byte, 0, len(strs))
	for _, it := range strs {
		if it == "" {
			continue
		}
		it = strings.TrimSpace(it)
		bs = append(bs, Parse2Byte(it))
	}
	return bs
}

func Parse2Byte(v string) byte {
	return s2i(v[2])<<4 + s2i(v[3])
}

func s2i(s byte) byte {
	if s <= 57 {
		return s - 48
	}
	// if s >= 97 {
	// }
	return s - 97 + 10
}

// extractFile extracts a FileDescriptorProto from a gzip'd buffer.
func ExtractFile(gz []byte) (*descriptor.FileDescriptorProto, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, fmt.Errorf("failed to open gzip reader: %v", err)
	}
	defer r.Close()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to uncompress descriptor: %v", err)
	}

	// fmt.Println(b)
	fd := new(descriptor.FileDescriptorProto)
	if err := proto.Unmarshal(b, fd); err != nil {
		return nil, fmt.Errorf("malformed FileDescriptorProto: %v", err)
	}

	return fd, nil
}

// extractFile extracts a FileDescriptorProto from a gzip'd buffer.
func ExtractGoGoFile(gz []byte) (*gogodescriptor.FileDescriptorProto, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, fmt.Errorf("failed to open gzip reader: %v", err)
	}
	defer r.Close()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to uncompress descriptor: %v", err)
	}

	// fmt.Println(b)
	fd := new(gogodescriptor.FileDescriptorProto)
	if err := gogoproto.Unmarshal(b, fd); err != nil {
		return nil, fmt.Errorf("malformed FileDescriptorProto: %v", err)
	}

	return fd, nil
}
