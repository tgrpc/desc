package desc

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/everfore/exc/walkexc"
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

func init() {
	associPB = make(map[string]*AssociPB, 100)
}

func Search(importName string) *AssociPB {
	a, _ := associPB[importName]
	return a
}

func AddPBGo(pg *PBGo) {
	iname := pg.ImportName()
	if _, ex := associPB[iname]; !ex {
		associPB[iname] = &AssociPB{
			PBs: make([]*PBGo, 0, 1),
		}
	}
	associPB[iname].PBs = append(associPB[iname].PBs, pg)
}

func Collect(dirs ...string) {
	walkexc.Setting(Cond, "")
	for _, dir := range dirs {
		d, err := absDir(dir)
		if err != nil {
			log.Errorf("dir: %s, err:%+v", d, err)
		}
		filepath.Walk(d, walkexc.WalkExc)
	}
}

func Cond(dir string, info os.FileInfo) (ifExec bool, skip error) {
	if strings.Contains(dir, ".git/") ||
		!strings.HasSuffix(info.Name(), ".pb.go") {
		return false, nil
	}
	if !info.IsDir() {
		AddPBGo(NewPBGo(dir, info.Name()))
		return true, nil
	}
	return false, nil
}
