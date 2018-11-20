package phpserialize

import "io"

// An Encoder writes PHP serialize values to an output stream.
type Encoder struct {
	w io.Writer
}

// Encode writes the PHP serialized value to the stream.
func (enc *Encoder) Encode(i interface{}) error {
	e := newEncodeState()
	err := e.marshal(i)
	if err != nil {
		return err
	}

	_, err = enc.w.Write(e.Bytes())
	return err
}

// NewEncoder returns a new encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}
