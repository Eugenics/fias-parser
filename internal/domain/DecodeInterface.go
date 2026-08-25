package domain

import "encoding/xml"

type DecoderInterface interface {
	Decode(decoder *xml.Decoder, se *xml.StartElement) (interface{}, error)
}
