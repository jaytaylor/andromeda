// Code generated by go-bindata.
// sources:
// index.tpl
// list.tpl
// package.tpl
// subpackage.tpl
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

var _packageTpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x56\x4f\x6f\xeb\x36\x0c\xbf\xe7\x53\xf0\x19\x39\x6c\x40\x13\xa3\xe8\x0e\x43\xe0\x18\x78\x6d\x50\xac\xc5\xd6\x17\x24\x5d\x77\x18\x76\x50\x6c\x26\x16\x62\x4b\x86\x24\x37\x35\x3c\x7f\xf7\x41\x7f\xec\xc8\x71\x3a\xf4\xd2\x9a\xe4\x8f\x14\xf9\x13\x49\xa5\x69\x60\x5a\x1e\x0f\xb0\x58\xc2\x1c\xda\x76\x12\x7d\x4b\x79\xa2\xea\x12\x21\x53\x45\x1e\x4f\xa2\xee\x1f\x92\x34\x9e\x00\x00\x44\x8a\xaa\x1c\xe3\xa6\x81\xf9\x9a\xa8\x0c\xda\x16\x66\xf0\x9d\xa5\x82\x17\x98\x92\x28\xb4\x66\x0b\x2d\x50\x11\x48\x32\x22\x24\xaa\x65\x50\xa9\xfd\xec\xd7\xc0\x37\x31\x52\xe0\x32\x20\x95\xca\xb8\x08\x20\xe1\x4c\x21\x53\xcb\xe0\x99\xd4\xf0\x4a\xea\x9c\x8b\x2b\xe8\x14\x65\x22\x68\xa9\x28\x67\x9e\x4b\x9f\x00\x50\x09\x04\x0e\x82\x94\x19\xf0\x3d\xa8\x0c\x01\x99\xa2\x02\xe1\xc8\xf8\x89\xc1\x3b\x95\x74\x97\x23\x1c\x38\x54\x8c\xbe\xa3\x90\x18\xc4\x93\x28\xb4\x05\x46\x3b\x9e\xd6\xf1\x64\x12\x65\xb7\xa6\xc2\x0d\x96\xfc\x85\x14\x08\x6d\x1b\x85\xd9\xad\xb6\xa4\xf4\x3d\x9e\x44\x04\x32\x81\xfb\x65\x10\x06\x71\x18\x85\x24\x9e\x34\x0d\x08\xc2\x0e\x08\xd3\x92\x08\x64\xca\x70\xa3\x59\x5d\xf7\xa2\x34\x04\xf7\x9e\x86\xfa\xde\xd8\x91\x19\xc4\x17\xfa\xfe\x78\x7b\x08\xb2\xd4\x84\x09\x6d\x1e\x99\xe8\x72\x7a\x5a\x2d\x40\xa7\xfc\xb4\x32\x68\x6b\xfe\x36\x9b\x45\xd9\x5d\xbc\x26\xc9\x91\x1c\x10\x36\x9c\x2b\x8b\x72\xa7\x45\x61\x76\x17\xcf\x66\xb1\x0d\xa1\xab\x95\x54\x71\x51\x3b\x64\x9f\xac\x76\xf9\x73\xf3\xbb\xce\x0f\x8c\x82\xf1\x3d\xcf\x73\x7e\x32\xe9\x3a\x93\x4e\xd1\x1d\xdc\x34\x70\xa2\x2a\x83\xe9\x09\x77\xda\xb8\x58\x9a\x36\x9b\xff\x65\xc5\xb6\xd5\x08\xba\x07\x86\x3d\xc4\xd8\x9d\xd1\x64\xb3\xe5\x95\x48\x10\x12\x9e\x22\x9c\x70\x07\x39\x65\xc7\x61\x4a\x9d\x6b\xc7\x5a\x2f\x0e\x33\x71\x94\x9d\xbf\x0c\x2f\x60\x0e\xd1\xec\x5a\x46\x7a\x9e\xb5\x1a\x0c\x27\x3d\xea\xc7\x89\xa1\xb0\x30\xf3\x79\x81\xd3\x5f\x6f\x0f\x5b\x0b\x78\x7b\xd8\x7a\x17\xa0\xff\x3e\x52\x21\x15\x48\x44\x66\x11\x46\xde\x22\xb2\xef\xea\x02\x79\x6e\x8e\xf7\xdb\xd0\xbb\xa6\x20\x7e\xde\xfe\x78\xf1\xaa\x1a\xa2\x33\xa5\x4a\xb9\x08\xc3\x03\x4f\x79\x32\xe7\xe2\x30\xf4\x5d\xf1\xa4\x2a\x90\x29\xa2\x67\xc6\x0f\x62\x9b\xc7\xeb\x8f\x27\xb6\xe7\xa2\x70\xb8\xec\x2e\x76\xb7\x34\x5f\x11\x45\xfa\x7b\xd1\x82\x19\x0c\x5b\x4d\x2f\x5e\xd4\x62\xf4\x0f\xbc\x28\xa8\x92\x1e\xd2\x69\xae\x81\xef\x05\x61\x49\x86\x3e\xba\x53\x5d\x83\xbf\x92\x83\x0f\xd5\xe2\xd5\xa8\xb5\x42\xf9\xca\x15\xc9\x3d\xf0\x5a\xa0\x52\xf5\xd9\x74\xcd\xf1\x91\x8b\xa3\x7f\x80\x91\xcf\xc0\x8e\x9b\xa7\xa2\xe4\x42\x61\x7a\x5f\x1b\x86\x34\xa7\x51\xf6\x4b\xdc\xa9\xe1\xbe\xd6\xe2\x24\xaa\x72\x7f\x49\xd0\xa2\xbc\x81\xa9\xc0\xbd\x34\x4b\x62\x18\xc4\xac\xbc\x9c\xc6\xc3\x5d\x41\x8b\xb2\xef\x74\xfb\x0d\x3f\x35\x0d\xe4\xc8\x6c\xa0\xf9\x46\x47\x6b\x5b\x3d\xa0\x28\x90\x25\x38\xb6\xfe\x0b\x65\x5e\x09\x92\x43\x10\x40\x20\x03\x68\xdb\x9f\x6d\x43\xe4\x74\xb8\x5d\x5c\xb6\xdd\xc0\xf4\xf3\x2c\xab\xdd\xda\x3e\x17\x94\xa5\xf8\xe1\xa8\xd9\x56\x3b\xd7\x42\x52\x47\xee\xc7\xdb\xc1\x5d\x79\xf2\x4c\x90\xeb\xbf\x8e\x26\x19\x85\x8e\x23\x90\xaa\xce\x71\x19\xe4\x54\xaa\x99\xf9\x9e\xe9\xd7\x68\x01\x8c\x33\xbd\xa8\x07\x0c\x9a\xb5\x32\x3e\xe2\xab\xf4\x7d\x5e\xf9\x68\x77\xb8\x7a\x08\x4b\x7b\x0a\xba\x83\x37\x48\x52\xb3\x39\xbc\xd2\xdc\x4c\x68\xc3\x02\xa2\x52\x98\xe7\x72\xe4\x10\x85\xda\xf2\x3f\x7b\x6a\x74\xfe\x98\xed\x73\xcb\xdd\xc5\xee\xba\xaf\x81\xe0\x05\xa5\x6e\x46\xa7\xfa\x14\x39\xee\x8f\x45\xbf\x08\x1c\xef\xb6\x0a\xbd\x5c\x6e\xfc\x76\xf0\xc3\xd8\xe9\xba\xf2\xda\x1d\x0f\xdd\x56\x0a\xcf\x84\x0c\x1e\xbe\x81\xca\xbc\x79\xd1\x4e\x7c\xb6\xc5\xbb\x59\x7b\x10\xe4\x94\xc3\x6f\x54\xea\xa7\xcb\xb6\x92\xd7\x27\xe9\xc7\x0d\x4c\x13\x03\xd1\x79\x3a\x58\xbf\xce\x9c\xfc\xb7\xe9\x8b\xf4\x03\xda\xf6\x1f\x58\x55\xc2\x2c\x41\x33\xfd\xd6\x77\xde\xe9\x2e\x56\xc5\x35\xf7\xad\x22\x7a\x98\x7d\xef\x67\xbe\x73\xda\xd1\xd2\xbf\x16\xe1\x91\x32\x2a\xb3\x51\x88\x4e\xfd\xa5\x18\xdb\x2a\x49\x10\xd3\x71\x1e\x9d\xfe\x0b\x31\xfe\x40\x29\xf5\x8d\x5e\x84\xe8\xd4\xe7\x08\xde\x04\xd9\x57\x25\xb4\xbf\xa4\xa2\xd0\xfe\x80\xfc\x2f\x00\x00\xff\xff\xb3\x06\x33\x7b\x68\x0a\x00\x00")

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

	info := bindataFileInfo{name: "package.tpl", size: 2664, mode: os.FileMode(436), modTime: time.Unix(1531356998, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _subpackageTpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x55\x4d\x6b\xeb\x38\x14\xdd\xfb\x57\xdc\x31\x6f\x39\xb1\x79\xb4\x8b\x21\x28\x86\xb6\xa1\x4c\x0a\x33\x84\xa6\xbb\x61\x16\x8a\x75\x63\x89\xd8\x92\x91\xe4\xb6\xc6\xe4\xbf\x0f\x92\xec\x44\x71\xc3\xd0\xc5\xdb\xb4\xf1\xd1\x3d\x47\xe7\x7e\xd9\xe4\x37\xa6\x4a\xdb\xb7\x08\xdc\x36\x75\x91\x90\xe9\x1f\x52\x56\x24\x00\x00\xc4\x0a\x5b\x63\x31\x0c\x90\x6d\xa9\xe5\x70\x3a\xc1\x02\x1e\x24\xd3\xaa\x41\x46\x49\x1e\x8e\x43\x68\x83\x96\x42\xc9\xa9\x36\x68\x57\x69\x67\x0f\x8b\x3f\xd2\xf8\x48\xd2\x06\x57\x29\xed\x2c\x57\x3a\x85\x52\x49\x8b\xd2\xae\xd2\x17\xda\xc3\x1b\xed\x6b\xa5\x6f\x44\x33\x34\xa5\x16\xad\x15\x4a\x46\x94\xb3\x01\x10\x06\x28\x54\x9a\xb6\x1c\xd4\x01\x2c\x47\x40\x69\x85\x46\x38\x4a\xf5\x21\xe1\x5d\x18\xb1\xaf\x11\x2a\x05\x9d\x14\xef\xa8\x0d\xa6\x45\x42\xf2\x90\x20\xd9\x2b\xd6\x17\x49\x42\xf8\xcf\x38\x43\x92\xf3\x9f\x0e\x65\xe2\xbd\x48\x08\x05\xae\xf1\xb0\x4a\xf3\xb4\xc8\x49\x4e\x8b\x64\x18\x40\x53\x59\x21\xfc\x68\xa9\x46\x69\x3d\x6b\xb9\x82\x6c\x7b\xac\xb2\xed\x19\x32\x67\xbd\x48\x63\x18\x62\xd6\x14\x90\x16\x33\xfc\x6f\xda\xa0\x37\x12\xae\x43\xc9\xbc\x4c\x1e\x1c\x71\xed\x3d\xdf\x15\xe1\x32\xd8\xd2\xf2\x48\x2b\x24\x39\xbf\x2b\x82\xeb\xcd\x7a\x09\x3e\xa1\x63\x95\x6d\xd6\x5e\x29\x50\xdd\xdf\x1d\xe2\x12\xae\x2c\x8d\xce\x2f\x5e\x62\xc0\x99\x88\xe9\x67\x26\xb7\xb6\x35\xcb\x3c\xaf\x14\x53\x65\xa6\x74\x95\x47\x35\x4c\x8b\xb5\x2a\xbb\x06\xa5\xa5\xae\x77\xb1\x48\x64\xdf\xfb\x86\x8d\x3c\x28\xdd\x8c\x71\x2e\x87\x61\x00\x71\x08\x26\xd6\xd4\x52\x9f\xbb\xe3\xba\x87\xec\x15\x5b\x75\xc9\xee\x0c\xcd\x92\xf4\xf8\x93\x6a\x1a\x61\xcd\x2c\x7a\x44\x6f\x11\x1e\x35\x95\x25\xc7\x39\x63\x82\x6f\x51\xde\x68\x35\x0f\x77\xd0\x4d\xf5\xde\xa2\x79\x53\x96\xd6\x33\xc2\x56\xa3\xb5\xfd\xe5\xf8\x16\xf9\x59\xe9\xe3\xfc\x22\x8f\x5d\x82\xa7\x89\xe5\xf7\xc5\xa6\x69\x95\xb6\x86\xe4\xfc\xbe\x48\x48\x57\x83\xb1\x7d\x8d\xab\xb4\x16\xc6\x2e\xfc\xef\x85\x5b\xfb\x25\x48\x25\xdd\x46\x5c\x66\x5a\x34\xad\x1f\xe6\x5d\xb7\xcf\x46\x15\x57\x7f\xbf\x97\xb5\x28\xae\x47\xd9\x05\x4f\xe3\x1b\x7e\x87\x46\xd7\xe2\x7a\x6e\x3b\xf7\x52\x09\x26\x2f\xf0\xd4\x66\x77\xd3\x2b\x52\xe6\x47\x3e\x8c\x47\xdc\x6d\x77\xb0\x04\xd2\xea\xf0\x16\xba\x8a\x26\xb9\x83\xbf\x28\x87\x11\x8b\xf5\x43\x26\xc8\x1e\xfb\x70\x3e\x55\x08\x19\x3c\xf6\x64\x2c\xd2\xff\x97\x61\x22\xff\x82\x4a\x38\x7b\x51\x1d\xdc\xfd\x4f\x9a\x7e\xd4\xf0\xa7\x30\x56\xe9\x3e\xb4\x2d\x32\xc3\x3e\x7f\x87\x1f\xa5\x0f\x71\xa6\xc6\xb0\xf3\x5e\x8c\xcf\xff\xf8\xcb\xd9\x27\x9c\x4e\xff\xc2\xba\xd3\x7e\xa3\xfc\xc8\x04\x6e\x36\x61\xb3\xf9\xba\x45\xdf\x59\xea\x52\x8e\xd9\x2f\x6a\x3f\xa2\x0f\xf6\x1b\x0a\xcf\x42\x0a\xc3\xbf\x48\x4c\xf0\xb7\x34\x76\x5d\x59\x22\xb2\xaf\x3e\x26\xfc\x1b\x1a\x7f\xa1\x31\xb4\x1a\x77\xfa\x22\x31\xc1\x91\xc2\xbc\x2d\x79\xf8\x3c\x90\x3c\x7c\x15\xff\x0b\x00\x00\xff\xff\x5c\x2e\x02\xd5\x2d\x07\x00\x00")

func subpackageTplBytes() ([]byte, error) {
	return bindataRead(
		_subpackageTpl,
		"subpackage.tpl",
	)
}

func subpackageTpl() (*asset, error) {
	bytes, err := subpackageTplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "subpackage.tpl", size: 1837, mode: os.FileMode(436), modTime: time.Unix(1531357019, 0)}
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
	"subpackage.tpl": subpackageTpl,
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
	"subpackage.tpl": &bintree{subpackageTpl, map[string]*bintree{}},
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

