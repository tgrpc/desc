package main

import (
	// "bufio"
	// "fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	// "io/ioutil"

	"github.com/everfore/exc/walkexc"
	// "github.com/tgrpc/desc"
	// "github.com/toukii/goutils"
	"git.ezbuy.me/ezbuy/base/misc/log"
)

type PBGo struct {
	AbsDir      string
	ParientName string
	Name        string
}

func NewPBGo(dir, name string) *PBGo {
	predir := path.Base(strings.TrimRight(dir, name))
	return &PBGo{
		AbsDir:      dir,
		ParientName: predir,
		Name:        name,
	}
}

func (pg *PBGo) ImportName() string {
	return path.Join(pg.ParientName, pg.Name)
}

// 相似pb
type AssociPB struct {
	PBs []*PBGo
}

var (
	associPB map[string]*AssociPB
)

func AddPBGo(pg *PBGo) {
	iname := pg.ImportName()
	if _, ex := associPB[iname]; !ex {
		associPB[iname] = &AssociPB{
			PBs: make([]*PBGo, 0, 1),
		}
	}
	associPB[iname].PBs = append(associPB[iname].PBs, pg)
}

func init() {
	associPB = make(map[string]*AssociPB, 100)
}

func main() {
	walkexc.Setting(Cond, "")
	filepath.Walk("$GOPATH/ezbuy/goflow/src/git.ezbuy.me/ezbuy/base", walkexc.WalkExc)
	log.JSON(associPB)
}

func Cond(dir string, info os.FileInfo) (ifExec bool, skip error) {
	if strings.Contains(dir, ".git/") ||
		!strings.HasSuffix(info.Name(), ".pb.go") {
		return false, nil
	}
	// fileSuffix := path.Ext(info.Name())
	if !info.IsDir() {
		// predir := path.Base(strings.TrimRight(dir, info.Name()))
		// fmt.Printf("%s : %s/%s \n", dir, predir, info.Name())

		pg := NewPBGo(dir, info.Name())
		AddPBGo(pg)

		return true, nil
	}
	return false, nil
}
