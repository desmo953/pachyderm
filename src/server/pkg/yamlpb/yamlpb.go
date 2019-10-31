package yamlpb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"gopkg.in/pachyderm/yaml.v3"
)

type Decoder struct {
	d *yaml.Decoder
}

func Marshal(v interface{}) ([]byte, error) {
	return MarshalTransform(v, nil)
}

func MarshalTransform(v interface{}, f func(map[string]interface{}) error) ([]byte, error) {
	// TODO(msteffen): Check if 'v' is a proto message and use jsonpb if so. This
	// would give us custom serialization for timestamps and such. The issue
	// preventing this is that the kubernetes API structs (e.g.
	// v1.ServiceAccount) contain embedded structs and use the `json:"inline"`
	// field annotation, but the gogo jsonpb library can't marshal embedded
	// structs.
	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	if err := e.Encode(v); err != nil {
		return nil, fmt.Errorf("serialization error while canonicalizing output: %v", err)
	}

	// Marshal to JSON first
	holder := map[string]interface{}{}
	if err := json.Unmarshal(buf.Bytes(), &holder); err != nil {
		return nil, fmt.Errorf("deserialization error while canonicalizing output: %v", err)
	}

	// transform 'holder' (e.g. de-stringifying TFJob)
	if f != nil {
		if err := f(holder); err != nil {
			return nil, err
		}
	}

	// Marshal to YAML
	bytes, err := yaml.Marshal(holder)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func Unmarshal(data []byte, v interface{}) error {
	d := NewDecoder(bytes.NewReader(data))
	return d.Decode(v)
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{d: yaml.NewDecoder(r)}
}

func (d *Decoder) Decode(v interface{}) error {
	return d.DecodeTransform(v, nil)
}

func (d *Decoder) DecodeTransform(v interface{}, f func(map[string]interface{}) error) error {
	holder := map[string]interface{}{}
	// deserialize yaml/json into 'holder'
	if err := d.d.Decode(&holder); err != nil {
		return fmt.Errorf("could not parse yaml: %v", err)
	}

	// transform 'holder' (e.g. stringifying TFJob)
	if f != nil {
		if err := f(holder); err != nil {
			return err
		}
	}

	// serialize 'holder' to json
	jsonBytes, err := json.Marshal(holder)
	if err != nil {
		return fmt.Errorf("serialization error while canonicalizing yaml: %v", err)
	}

	// parse again into 'v', with special parser
	if msg, ok := v.(proto.Message); ok {
		decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
		if err := jsonpb.UnmarshalNext(decoder, msg); err != nil {
			return fmt.Errorf("error canonicalizing yaml while parsing to proto: %v", err)
		}
	} else {
		if err := json.Unmarshal(jsonBytes, v); err != nil {
			return fmt.Errorf("parse error while canonicalizing yaml: %v", err)
		}
	}
	return nil
}
