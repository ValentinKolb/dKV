package serializer

import (
	"bytes"
	"encoding/gob"
	"github.com/ValentinKolb/dKV/rpc/common"
)

// NewGOBSerializer creates a new serializer using Go's binary gob format
func NewGOBSerializer() IRPCSerializer {
	return &gobSerializerImpl{}
}

// gobSerializerImpl implements the IRPCSerializer interface using gob encoding
type gobSerializerImpl struct {
}

// --------------------------------------------------------------------------
// Interface Methods (docu see serializer.IRPCSerializer)
// --------------------------------------------------------------------------

func (g gobSerializerImpl) Serialize(msg common.Message) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(msg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g gobSerializerImpl) Deserialize(b []byte, msg *common.Message) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	return dec.Decode(msg)
}
