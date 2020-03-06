package desc

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/everfore/exc/walkexc"
	"github.com/toukii/goutils"
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

func SearchByFilename(filename string) *AssociPB {
	apb := &AssociPB{
		PBs: make([]*PBGo, 0, 1),
	}
	for name, it := range associPB {
		fmt.Println(name, filename)
		if strings.HasSuffix(name, filename) {
			apb.PBs = append(apb.PBs, it.PBs...)
		}
	}
	return apb
}

func SearchByMethod(packageName, serviceName, method string, unique map[string]bool) *AssociPB {
	apb := &AssociPB{
		PBs: make([]*PBGo, 0, 1),
	}
	for _, it := range associPB {
		for _, pb := range it.PBs {
			if unique[pb.ImportName()] {
				continue
			}
			if pb.ParientName == packageName {
				pbfilebs := goutils.ReadFile(pb.AbsDir)
				pbfilecnt := goutils.ToString(pbfilebs)
				if strings.Contains(pbfilecnt, method) {
					apb.PBs = append(apb.PBs, pb)
				}
			}
		}
	}
	return apb
}

func SearchByImportFilename(importfilename string) *AssociPB {
	a, _ := associPB[importfilename]
	if a == nil {
		return SearchByFilename(importfilename)
	}
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
