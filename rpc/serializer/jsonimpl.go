package serializer

import (
	"encoding/json"
	"github.com/ValentinKolb/dKV/rpc/common"
)

// NewJSONSerializer creates a new serializer using json encoding
func NewJSONSerializer() IRPCSerializer {
	return &jsonSerializerImpl{}
}

// jsonSerializerImpl implements the IRPCSerializer interface using json encoding
type jsonSerializerImpl struct {
}

// --------------------------------------------------------------------------
// Interface Methods (docu see serializer.IRPCSerializer)
// --------------------------------------------------------------------------

func (j jsonSerializerImpl) Serialize(msg common.Message) ([]byte, error) {
	return json.Marshal(msg)
}

func (j jsonSerializerImpl) Deserialize(b []byte, msg *common.Message) error {
	return json.Unmarshal(b, msg)
}
