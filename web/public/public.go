// Code generated by go-bindata.
// sources:
// index.tpl
// list.tpl
// package.tpl
// DO NOT EDIT!

package public

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _indexTpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x84\x93\xd1\x6a\xe3\x3a\x10\x86\xef\xf3\x14\x73\x4c\x6f\x6b\xd3\xf6\xe6\x50\x64\x43\x36\x85\x85\xa5\xbb\x14\xda\x7d\x80\x89\x35\xb1\x44\x64\xc9\x3b\x1a\x27\x1b\x8c\xdf\x7d\x91\xed\xa4\x69\x59\x58\xdf\x58\xcc\x7c\xfa\x47\xff\x8c\xa4\xfe\xd3\xa1\x96\x53\x47\x60\xa4\x75\xd5\x4a\x9d\x7f\x84\xba\x5a\x01\x00\x28\xb1\xe2\xa8\x5a\x7b\xcd\xa1\x25\x8d\xaa\x98\x03\x73\xb2\x25\x41\xa8\x0d\x72\x24\x29\xb3\x5e\x76\xb7\xff\x67\xd7\x29\x8f\x2d\x95\x19\xf6\x62\x02\x67\x50\x07\x2f\xe4\xa5\xcc\xbe\xe1\x09\xde\xf0\xe4\x02\xff\x85\xd6\x14\x6b\xb6\x9d\xd8\xe0\xaf\xb6\x5c\x0e\x00\x36\x02\x42\xc3\xd8\x19\x08\x3b\x10\x43\x40\x5e\x2c\x13\xec\x7d\x38\x7a\x38\xd8\x68\xb7\x8e\xa0\x09\xd0\x7b\x7b\x20\x8e\x94\x55\x2b\x55\xcc\x96\xd4\x36\xe8\x53\x32\x78\x77\x6d\xc9\xdc\xa5\xd0\x7d\xf5\x4a\xc8\xb5\xb9\xd6\x3c\xab\x7d\x0d\x0e\x7d\x03\x3f\x17\x45\x55\x98\xfb\x24\xc6\xd5\x6a\x18\xe0\x68\xc5\xc0\x8d\x43\xa1\x28\xf0\x58\x42\xbe\x09\x7e\x67\x9b\xfc\x3b\x46\x21\xce\x9f\xe7\xc4\x38\x26\xd6\xee\x2e\xe4\x38\xae\x94\xb6\x87\xa5\x03\xe6\xa1\x5a\xc0\x0d\xe3\xd1\x91\x86\x17\xac\xf7\xd8\x50\x54\x85\x79\x58\xa0\xde\x41\x94\x93\xa3\x32\x73\x36\xca\xed\xb4\xbe\x4d\xf3\x7b\x04\x1f\x7c\x32\x3a\x0c\xc0\xe8\x1b\x82\x9b\x6e\xdf\xa4\xc3\x5c\x55\x83\xe5\x53\xce\x56\xc3\x30\x11\xf9\x13\x0a\xe6\x1b\x26\x14\xd2\xeb\x44\x81\x42\x30\x4c\xbb\x32\x2b\xce\xcc\x0b\x8a\x81\x71\xcc\xaa\x4f\x01\x55\x60\xa5\x0a\x67\xa7\xaa\xe4\xf5\xb9\x86\x2a\xfa\x74\x89\x8a\xc9\xdc\x7b\xea\x7d\xf5\xa3\x6f\xb7\xc4\x69\x7c\xdd\xe2\x11\xac\x07\xeb\x35\xfd\x7e\x84\x61\x80\xfc\xe9\x4b\x7e\x76\xff\x4c\x7e\xea\x54\xea\xf5\xd4\x19\xf8\xd5\x53\x4f\xe0\xc8\x37\x62\x2e\xf8\x5b\x98\x92\x1f\xf0\x75\x2d\xf6\x40\x70\x0c\xbc\x27\x8e\x13\x3a\xd5\xf8\x3c\xa0\x57\x41\x89\x90\x31\xb5\x41\x28\x66\x1f\xcb\x45\x88\xd6\xd7\x04\x1d\x87\x86\xb1\x85\x28\xc8\x42\xfa\x9f\x6a\xf5\xb4\x79\x16\x2b\x96\x5b\x57\xcc\xcf\xeb\x4f\x00\x00\x00\xff\xff\x0f\x9f\x77\x2f\x76\x03\x00\x00")

func indexTplBytes() ([]byte, error) {
	return bindataRead(
		_indexTpl,
		"index.tpl",
	)
}

func indexTpl() (*asset, error) {
	bytes, err := indexTplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "index.tpl", size: 886, mode: os.FileMode(436), modTime: time.Unix(1529878619, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _listTpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x90\x41\x6b\xc3\x30\x0c\x85\xef\xfe\x15\x6f\x66\xc7\x75\xb9\x8e\xe1\x18\x76\xdd\xa9\x87\xfd\x01\x35\x56\x63\xd3\x44\x0e\x8e\xda\x11\x4c\xfe\xfb\x68\xc3\x4a\x07\x3b\xd9\x42\xef\x7b\xd2\x93\x7b\x0a\xb9\xd3\x65\x62\x44\x1d\x07\x6f\xdc\xef\xc3\x14\xbc\x01\x00\xa7\x49\x07\xf6\xb5\x62\x60\xc1\x2b\xd6\x15\x7b\xea\x4e\xd4\xf3\x8c\x1d\x3e\x24\x94\x3c\x72\x20\xd7\x6c\xba\x8d\x19\x59\x09\x5d\xa4\x32\xb3\xb6\xf6\xac\xc7\xdd\x9b\x7d\x6c\x09\x8d\xdc\x5a\x3a\x6b\xcc\xc5\xa2\xcb\xa2\x2c\xda\xda\x4f\x5a\xf0\x45\xcb\x90\xcb\x3f\xea\xc0\x73\x57\xd2\xa4\x29\xcb\x03\x72\x5f\x00\x69\x06\xa1\x2f\x34\x45\xe4\x23\x34\x32\x58\x34\x15\xc6\x49\xf2\xb7\xe0\x92\xe6\x74\x18\x18\x7d\xc6\x59\xd2\x85\xcb\xcc\xd6\x1b\xd7\x6c\x49\xdd\x21\x87\xc5\x1b\x53\x2b\x0a\x49\xcf\x78\x9e\x4e\xfd\x9e\x34\xbe\xdc\x7e\x78\x6f\x6f\xd1\x8d\x23\xc4\xc2\xc7\xd6\x36\xb5\xde\x35\x58\x57\xeb\xff\xd6\xae\xa1\xab\x69\xf1\x57\x47\x96\x70\x65\x8d\x6b\xb6\x29\xae\xd9\xae\xfc\x13\x00\x00\xff\xff\x59\xb1\xc1\x65\x7d\x01\x00\x00")

func listTplBytes() ([]byte, error) {
	return bindataRead(
		_listTpl,
		"list.tpl",
	)
}

func listTpl() (*asset, error) {
	bytes, err := listTplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "list.tpl", size: 381, mode: os.FileMode(436), modTime: time.Unix(1529871495, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _packageTpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x56\xc1\x6e\xe3\x38\x0c\xbd\xe7\x2b\x38\x46\x8f\x93\x18\x45\xf7\xb0\x08\x14\x03\xd3\x06\xc5\xb6\x98\xed\x14\xc9\xcc\x5c\x16\x7b\x50\x2c\x26\x12\x6a\x4b\x86\x24\x37\x35\x0c\xff\xfb\x40\x92\xed\xca\xa9\x0f\xbd\xd4\x22\xf9\x48\x3e\x52\x14\x1b\xf2\x85\xa9\xdc\x36\x15\x02\xb7\x65\x91\x2d\xc8\xf0\x41\xca\xb2\x05\x00\x00\xb1\xc2\x16\x98\xb5\x2d\xac\x9e\xa9\xe5\xd0\x75\xb0\x84\x6f\x92\x69\x55\x22\xa3\x24\x0d\xe6\x00\x2d\xd1\x52\xc8\x39\xd5\x06\xed\x26\xa9\xed\x71\xf9\x77\x12\x9b\x24\x2d\x71\x93\xd0\xda\x72\xa5\x13\xc8\x95\xb4\x28\xed\x26\x79\xa4\x0d\xfc\xa4\x4d\xa1\xf4\x0c\x9a\xa1\xc9\xb5\xa8\xac\x50\x32\x72\x19\x09\x80\x30\x40\xe1\xa4\x69\xc5\x41\x1d\xc1\x72\x04\x94\x56\x68\x84\x17\xa9\xce\x12\x5e\x85\x11\x87\x02\xe1\xa4\xa0\x96\xe2\x15\xb5\xc1\x24\x5b\x90\x34\x14\x48\x0e\x8a\x35\xd9\x62\x41\xf8\xb5\xaf\x70\x87\x95\x7a\xa2\x25\x42\xd7\x91\x94\x5f\x3b\x0b\x13\xaf\xd9\x82\x50\xe0\x1a\x8f\x9b\x24\x4d\xb2\x94\xa4\x34\x5b\xb4\x2d\x68\x2a\x4f\x08\x57\x15\xd5\x28\xad\xef\xcd\x7a\xe3\x9a\x34\x88\x06\xba\x2e\xf2\x6c\xdb\x18\x3b\x34\x33\xc9\x2e\xf4\x63\xfa\x90\x04\x25\xf3\x61\xd2\xc0\x83\xeb\x81\xd3\xc3\x76\x0d\x8e\xf2\xc3\xd6\xa3\x83\xf9\xcb\x72\x49\xf8\x4d\xf6\x4c\xf3\x17\x7a\x42\xd8\x29\x65\x03\xaa\xcf\x46\x52\x7e\x93\x2d\x97\x59\x08\xf1\x6b\xf7\x7d\x0d\x23\x41\x07\xfb\xb5\xfb\xee\x38\x81\x57\x48\x75\x54\x45\xa1\xce\x9e\x62\x6f\x72\xb4\xa2\x64\xe0\xe3\x38\xca\x21\xcd\x48\xde\xa9\xc1\x27\x1a\x51\x3f\xce\x12\x75\x80\xf9\xe3\x05\xce\x9d\x7e\xdf\xed\x03\xe0\xf7\xdd\x3e\xaa\xca\xfd\xbd\x17\xda\x58\x30\x88\x32\x20\xbc\xbc\x47\x94\xdf\xec\x05\xf2\xbd\xe3\xaf\xd7\x69\x54\x7b\x92\x3d\xee\x7f\x3c\xc5\x05\x4c\xd0\xdc\xda\xca\xac\xd3\xf4\xa4\x98\xca\x57\x4a\x9f\xa6\xbe\x5b\x95\xd7\x25\x4a\x4b\xdd\x20\xc6\x41\xc2\x8d\x44\x4d\x7f\x90\x47\xa5\xcb\x1e\xc7\x6f\xfc\x2d\x8a\x23\xac\xb6\xd4\x52\x7f\x95\xce\xcf\x09\x7e\xda\x42\x35\xa3\x78\x51\x8b\xd7\xdf\xa9\xb2\x14\xd6\x44\xc8\x5e\x33\x07\xbe\xd5\x54\xe6\x1c\x63\xf4\xa0\x9a\x83\xff\xa4\xa7\x18\xea\xc4\xd9\xa8\x8d\x9d\x86\x74\xf2\x1c\xf0\x5e\xe9\x97\x18\xe8\xe5\x77\x60\xdb\xc2\x59\x58\x0e\x57\xa6\x3e\x3c\xbf\x9c\xdc\x83\x11\x92\xe1\x5b\x8f\xde\xd7\x87\xbe\x8b\x06\x92\xc4\x35\x2b\xf4\x8e\x4a\x36\xba\xf4\xdf\xd5\x0e\x29\xf3\xc3\x16\xae\x20\xee\xaa\x33\xac\x81\x54\xda\xaf\xad\x0f\x0e\x24\x75\x96\x77\x46\xfd\x13\x9b\x3b\x85\xdb\x8d\x2e\x70\x42\xd1\x03\x6e\x5c\x8e\x02\xe5\xac\x1d\x9e\xd0\x58\x64\x30\xa8\xd6\xe3\x44\xf4\xcb\x23\x70\x73\x53\xf6\x35\x6e\x4a\x1c\xe6\x59\xa3\xb5\x4d\xcf\x2b\x72\xf0\xe9\x0f\x7a\xbe\x84\x81\xf3\x43\x59\x29\x6d\x91\xdd\x36\x81\xed\x5f\xd9\xa0\x81\xdb\xc6\x89\x0b\x52\x17\x31\x21\x51\x56\x9e\xc0\xd4\xd1\x2f\xe5\x42\x64\xd3\x6d\xe6\xb0\xc3\x06\x0b\xe7\xf0\x30\x0a\x31\x5d\x5d\x75\xd1\xbf\x93\x88\x9f\xcb\x7d\xa7\xe9\xb9\x80\x7f\x84\xb1\x4a\x37\x24\x75\x6c\x22\x22\xec\xed\x2b\x5c\xe5\x1e\xe2\x08\xf5\xb0\xf1\x05\xf5\xf2\x7f\x3e\x39\x7b\x83\xae\xfb\x1f\xb6\xb5\xf6\xef\xce\x0f\x60\xf0\x5d\x0d\xba\x8b\x69\x9d\x73\xdf\x5b\xea\x4a\x8e\xbd\x1f\xd5\xa1\xd7\x7e\xd8\x33\x73\x11\xee\x85\x14\x86\x7f\x08\x31\xa8\x3f\x15\x63\x5f\xe7\x39\x22\xfb\xc8\x63\xd0\x7f\x22\xc6\xbf\x68\x8c\x9f\xb7\x69\x88\x41\x1d\x45\xb8\xbc\x96\x34\xfc\x47\x24\x69\xf8\x21\xf0\x27\x00\x00\xff\xff\x03\x82\x07\xd6\x20\x08\x00\x00")

func packageTplBytes() ([]byte, error) {
	return bindataRead(
		_packageTpl,
		"package.tpl",
	)
}

func packageTpl() (*asset, error) {
	bytes, err := packageTplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "package.tpl", size: 2080, mode: os.FileMode(436), modTime: time.Unix(1529881801, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"index.tpl": indexTpl,
	"list.tpl": listTpl,
	"package.tpl": packageTpl,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"index.tpl": &bintree{indexTpl, map[string]*bintree{}},
	"list.tpl": &bintree{listTpl, map[string]*bintree{}},
	"package.tpl": &bintree{packageTpl, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

