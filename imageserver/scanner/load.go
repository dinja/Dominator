package scanner

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/concurrent"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/objectserver"
	"io"
	"log"
	"os"
	"path"
	"syscall"
	"time"
)

func loadImageDataBase(baseDir string, objSrv objectserver.ObjectServer,
	logger *log.Logger) (*ImageDataBase, error) {
	fi, err := os.Stat(baseDir)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Cannot stat: %s\t%s\n", baseDir, err))
	}
	if !fi.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory\n", baseDir))
	}
	imdb := new(ImageDataBase)
	imdb.baseDir = baseDir
	imdb.imageMap = make(map[string]*image.Image)
	imdb.addNotifiers = make(notifiers)
	imdb.deleteNotifiers = make(notifiers)
	imdb.objectServer = objSrv
	imdb.logger = logger
	state := concurrent.NewState(0)
	startTime := time.Now()
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	if err := imdb.scanDirectory("", state); err != nil {
		return nil, err
	}
	if err := state.Reap(); err != nil {
		return nil, err
	}
	if logger != nil {
		plural := ""
		if imdb.CountImages() != 1 {
			plural = "s"
		}
		syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop)
		userTime := time.Duration(rusageStop.Utime.Sec)*time.Second +
			time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
			time.Duration(rusageStart.Utime.Sec)*time.Second -
			time.Duration(rusageStart.Utime.Usec)*time.Microsecond
		logger.Printf("Loaded %d image%s in %s (%s user CPUtime)\n",
			imdb.CountImages(), plural, time.Since(startTime), userTime)
	}
	return imdb, nil
}

func (imdb *ImageDataBase) scanDirectory(dirname string,
	state *concurrent.State) error {
	file, err := os.Open(path.Join(imdb.baseDir, dirname))
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	for _, name := range names {
		filename := path.Join(dirname, name)
		var stat syscall.Stat_t
		err := syscall.Lstat(path.Join(imdb.baseDir, filename), &stat)
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
		if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			err = imdb.scanDirectory(filename, state)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
			err = state.GoRun(func() error {
				if err := imdb.loadFile(filename); err != nil {
					return fmt.Errorf("cannot load image: %s: %s", filename, err)
				}
				return nil
			})
		}
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
	}
	return nil
}

func (imdb *ImageDataBase) loadFile(filename string) error {
	file, err := os.Open(path.Join(imdb.baseDir, filename))
	if err != nil {
		return err
	}
	defer file.Close()
	reader := fsutil.NewChecksumReader(file)
	decoder := gob.NewDecoder(reader)
	var image image.Image
	if err := decoder.Decode(&image); err != nil {
		return err
	}
	if err := reader.VerifyChecksum(); err != nil {
		if err != io.EOF {
			return err
		}
	}
	image.FileSystem.RebuildInodePointers()
	imdb.Lock()
	defer imdb.Unlock()
	imdb.imageMap[filename] = &image
	return nil
}
