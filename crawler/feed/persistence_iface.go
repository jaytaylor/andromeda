package feed

type Persistence interface {
	MetaSave(key string, src interface{}) error // Store metadata key/value.  NB: src must be one of raw []byte, string, or proto.Message struct.
	MetaDelete(key string) error                // Delete a metadata key.
	Meta(key string, dst interface{}) error     // Retrieve metadata key and populate into dst.  NB: dst must be one of *[]byte, *string, or proto.Message struct.
}
