package decoder

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"runtime"
	"time"

	"github.com/vmihailenco/msgpack"
)

// FBitDecoder handles decoding of fluent-bit msgpack messages.
type FBitDecoder struct {
	r        io.Reader
	msgpkdec *msgpack.Decoder
}

func init() {
	msgpack.RegisterExt(0, (*FBTime)(nil))
}

// FBTime is used to unmarshall fluent-bit unix timestamps.
type FBTime struct {
	time.Time
}

// UnmarshalMsgpack converts a msgpack-encoded fluent-bit Unix timestamp to a golang unix timestamp.
func (fb *FBTime) UnmarshalMsgpack(b []byte) error {
	if len(b) != 8 {
		return errors.New("Invalid timestamp data")
	}

	var usec uint32
	sec := binary.BigEndian.Uint32(b)
	if runtime.GOOS == "darwin" {
		usec = binary.LittleEndian.Uint32(b[4:])
	} else {
		usec = binary.BigEndian.Uint32(b[4:])
	}
	fb.Time = time.Unix(int64(sec), int64(usec))
	return nil
}

var _ msgpack.Unmarshaler = (*FBTime)(nil)

// NewDecoder takes the provided io.Reader with a messagepack-encoded fluent-bit message
// and returns a pre-configured FBitDecoder.
func NewDecoder(r io.Reader) *FBitDecoder {
	dec := new(FBitDecoder)
	dec.r = r
	dec.msgpkdec = msgpack.NewDecoder(r)

	return dec
}

// NewDecoderBytes takes the provides []byte input and returns a preconfigured FBitDecoder.
func NewDecoderBytes(in []byte) *FBitDecoder {
	dec := new(FBitDecoder)
	return dec
}

// GetRecord returns a single messages from the payload. Caller should this func until an EOF error is returned.
// EOFs denote there are no more records in this payload and decoding is complete.
func (dec *FBitDecoder) GetRecord() (interface{}, error) {
	data, err := dec.msgpkdec.DecodeInterface()
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ExtractTime returns the FBTime of the current record.
func ExtractTime(d interface{}) (FBTime, error) {
	slice := reflect.ValueOf(d)
	if slice.Kind() != reflect.Slice {
		return FBTime{}, errors.New("Unknown Data")
	}

	t := slice.Index(0).Interface()

	return t.(FBTime), nil
}

// ExtractData returns a map[string]interface{} of the record data.
func ExtractData(d interface{}) (map[string]interface{}, error) {
	slice := reflect.ValueOf(d)
	if slice.Kind() != reflect.Slice {
		return nil, errors.New("Unknown Data")
	}

	t := slice.Index(1).Interface()
	return t.(map[string]interface{}), nil
}
