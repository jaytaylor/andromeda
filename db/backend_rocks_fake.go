// +build !rocks

package db

const rocksSupportAvailable = false

type RocksConfig struct {
	Dir string
}

func NewRocksConfig(_ string) *RocksConfig {
	return &RocksConfig{}
}

func (cfg RocksConfig) Type() Type {
	return Rocks
}

type RocksBackend struct{}

func NewRocksBackend(_ *RocksConfig) *RocksBackend {
	return &RocksBackend{}
}

func (_ *RocksBackend) Open() error                   { return ErrBackendBuildNotSupported }
func (_ *RocksBackend) Close() error                  { return ErrBackendBuildNotSupported }
func (_ *RocksBackend) New(_ string) (Backend, error) { return nil, ErrBackendBuildNotSupported }
func (_ *RocksBackend) Get(_ string, _ []byte) ([]byte, error) {
	return nil, ErrBackendBuildNotSupported
}
func (_ *RocksBackend) Put(_ string, _ []byte, _ []byte) error   { return ErrBackendBuildNotSupported }
func (_ *RocksBackend) Delete(_ string, _ ...[]byte) error       { return ErrBackendBuildNotSupported }
func (_ *RocksBackend) Destroy(_ ...string) error                { return ErrBackendBuildNotSupported }
func (_ *RocksBackend) Len(_ string) (int, error)                { return 0, ErrBackendBuildNotSupported }
func (_ *RocksBackend) Begin(_ bool) (Transaction, error)        { return nil, ErrBackendBuildNotSupported }
func (_ *RocksBackend) View(_ func(_ Transaction) error) error   { return ErrBackendBuildNotSupported }
func (_ *RocksBackend) Update(_ func(_ Transaction) error) error { return ErrBackendBuildNotSupported }
func (_ *RocksBackend) EachRow(_ string, _ func(_ []byte, _ []byte)) error {
	return ErrBackendBuildNotSupported
}
func (_ *RocksBackend) EachRowWithBreak(_ string, _ func(key []byte, value []byte) bool) error {
	return ErrBackendBuildNotSupported
}
func (_ *RocksBackend) EachTable(_ func(_ string, _ Transaction) error) error {
	return ErrBackendBuildNotSupported
}
