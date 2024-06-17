package telemetry

import "encoding/json"

// textMap is a wrapper around map[string]string to conform to the propagation.TextMapCarrier
// interface required by OpenTelemetry to help serialize trace state into a string that can
// be passed around manually
type textMap map[string]string

// newTextMap return a new TextMap.
func newTextMap() textMap {
	return textMap{}
}

// newTextMapFromSerialized returns a TextMap based on the incoming serialized TextMap bytes.
func newTextMapFromSerialized(serialized []byte) (textMap, error) {
	tm := newTextMap()
	err := json.Unmarshal(serialized, &tm)
	if err != nil {
		return tm, err
	}
	return tm, nil
}

// Get returns a value from the TextMap.
func (tm textMap) Get(key string) string {
	return tm[key]
}

// Set sets a value in the TextMap.
func (tm textMap) Set(key string, value string) {
	tm[key] = value
}

// Keys returns the keys of the TextMap.
func (tm textMap) Keys() []string {
	keys := make([]string, 0, len(tm))
	for k := range tm {
		keys = append(keys, k)
	}
	return keys
}

// Serialize serializes the TextMap into JSON bytes.
func (tm textMap) Serialize() ([]byte, error) {
	return json.Marshal(tm)
}
