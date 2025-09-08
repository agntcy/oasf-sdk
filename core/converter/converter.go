package converter

import (
	corev1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/core/v1"
)

var DefaultConverter Converter = &converter{}

type Converter interface {
	Decode(in *corev1.EncodedRecord) (out *corev1.DecodedRecord, err error)
	Encode(in *corev1.DecodedRecord) (out *corev1.EncodedRecord, err error)
}

func Decode(in *corev1.EncodedRecord) (*corev1.DecodedRecord, error) {
	return DefaultConverter.Decode(in)
}

func Encode(in *corev1.DecodedRecord) (*corev1.EncodedRecord, error) {
	return DefaultConverter.Encode(in)
}

type converter struct{}

func (c *converter) Decode(in *corev1.EncodedRecord) (*corev1.DecodedRecord, error) {
	panic("not implemented")
}

func (c *converter) Encode(in *corev1.DecodedRecord) (*corev1.EncodedRecord, error) {
	panic("not implemented")
}
