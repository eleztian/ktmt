package ktmt

import (
	"git.moresec.cn/zhangtian/ktmt/packets"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"sync"
)

const (
	msgExt     = ".msg"
	tmpExt     = ".tmp"
	corruptExt = ".CORRUPT"
)

type fileStore struct {
	sync.RWMutex
	directory string
	opened    bool
}

func NewFileStore(directory string) Store {
	store := &fileStore{
		directory: directory,
		opened:    false,
	}
	return store
}

func (f *fileStore) Open() {
	f.Lock()
	defer f.Unlock()
	// if no store directory was specified in ClientOpts, by default use the
	// current working directory
	if f.directory == "" {
		f.directory, _ = os.Getwd()
	}

	// if store dir exists, great, otherwise, create it
	if !exists(f.directory) {
		perms := os.FileMode(0770)
		merr := os.MkdirAll(f.directory, perms)
		chkerr(merr)
	}
	f.opened = true
}

func (f *fileStore) Close() {
	f.Lock()
	defer f.Unlock()

	f.opened = false
}

func (f *fileStore) all() []string {
	var err error
	var keys []string
	var files fileInfos

	if !f.opened {
		return nil
	}

	files, err = ioutil.ReadDir(f.directory)
	chkerr(err)
	sort.Sort(files)
	for _, f := range files {
		name := f.Name()
		if name[len(name)-len(msgExt):] != msgExt {
			continue
		}
		key := name[0 : len(name)-4] // remove file extension
		keys = append(keys, key)
	}
	return keys
}

func (f *fileStore) Reset() {
	f.Lock()
	defer f.Unlock()

	all := f.all()
	for _, k := range all {
		f.del(k)
	}

}

func (f *fileStore) Put(key string, message packets.ControlPacket) {
	f.Lock()
	defer f.Unlock()
	if !f.opened {
		return
	}
	message.SetDup()
	fileWrite(f.directory, key, message)
}

func (f *fileStore) Get(key string) packets.ControlPacket {
	f.Lock()
	defer f.Unlock()

	if !f.opened {
		return nil
	}

	filepath := fullpath(f.directory, key)
	if !exists(filepath) {
		return nil
	}

	file, err := os.Open(filepath)
	chkerr(err)

	msg, rerr := packets.ReadPacket(file)
	cerr := file.Close()
	chkerr(cerr)

	if rerr != nil {
		newpath := corruptpath(f.directory, key)
		_ = os.Rename(filepath, newpath)
		return nil
	}

	return msg
}

func (f *fileStore) All() []string {
	f.Lock()
	defer f.Unlock()

	return f.all()
}

func (f *fileStore) del(key string) {
	if !f.opened {
		return
	}

	filepath := fullpath(f.directory, key)
	if !exists(filepath) {
		return
	}

	err := os.Remove(filepath)
	chkerr(err)
}

func (f *fileStore) Del(key string) {
	f.Lock()
	defer f.Unlock()

	f.del(key)
}

func exists(file string) bool {
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

type fileInfos []os.FileInfo

func (f fileInfos) Len() int {
	return len(f)
}

func (f fileInfos) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f fileInfos) Less(i, j int) bool {
	return f[i].ModTime().Before(f[j].ModTime())
}

func fullpath(store string, key string) string {
	p := path.Join(store, key+msgExt)
	return p
}

func tmppath(store string, key string) string {
	p := path.Join(store, key+tmpExt)
	return p
}

func corruptpath(store string, key string) string {
	p := path.Join(store, key+corruptExt)
	return p
}

func fileWrite(store, key string, m packets.ControlPacket) {
	temppath := tmppath(store, key)
	f, err := os.Create(temppath)
	chkerr(err)
	werr := m.Write(f)
	chkerr(werr)
	cerr := f.Close()
	chkerr(cerr)
	rerr := os.Rename(temppath, fullpath(store, key))
	chkerr(rerr)
}
