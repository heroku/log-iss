package decoder

import (
	"encoding/binary"
	"io"
	"reflect"
	"time"

	"github.com/ugorji/go/codec"
)

// FBitDecoder handles decoding of fluent-bit msgpack messages.
type FBitDecoder struct {
	handle *codec.MsgpackHandle
	mpdec  *codec.Decoder
}

// FLBTime is a custom type used by codec to decode unix timestamps.
type FLBTime struct {
	time.Time
}

// WriteExt is unsupported
func (f FLBTime) WriteExt(interface{}) []byte {
	panic("unsupported")
}

// ReadExt powers the FLBTime conversion during codec decoding.
func (f FLBTime) ReadExt(i interface{}, b []byte) {
	out := i.(*FLBTime)
	sec := binary.BigEndian.Uint32(b)
	// TODO(mt): This is really weird, but the data looks... interesting. See https://gist.github.com/amerine/322f9c368f9fc0e9dc8d74f2e6b59bcf
	// usec := binary.BigEndian.Uint32(b[4:])
	usec := binary.LittleEndian.Uint32(b[4:])

	out.Time = time.Unix(int64(sec), int64(usec))
}

// ConvertExt noop.
func (f FLBTime) ConvertExt(v interface{}) interface{} {
	return nil
}

// UpdateExt is unsupported.
func (f FLBTime) UpdateExt(dest interface{}, v interface{}) {
	panic("unsupported")
}

// NewDecoder takes the provided io.Reader with a messagepack-encoded fluent-bit message
// and returns a pre-configured FBitDecoder.
func NewDecoder(r io.Reader) *FBitDecoder {
	dec := new(FBitDecoder)
	dec.handle = new(codec.MsgpackHandle)
	dec.handle.RawToString = true
	dec.handle.SetExt(reflect.TypeOf(FLBTime{}), 0, &FLBTime{})
	dec.mpdec = codec.NewDecoder(r, dec.handle)

	return dec
}

// GetRecord returns a single messages from the payload.
func GetRecord(dec *FBitDecoder) (ret int, ts interface{}, rec map[interface{}]interface{}) {
	var m interface{}

	err := dec.mpdec.Decode(&m)
	if err != nil {
		return -1, 0, nil
	}

	slice := reflect.ValueOf(m)
	if slice.Kind() != reflect.Slice {
		// Not a fluent-bit message
		return -1, 0, nil
	}
	t := slice.Index(0).Interface()
	data := slice.Index(1)

	mapdata := data.Interface().(map[interface{}]interface{})

	return 0, t, mapdata
}
