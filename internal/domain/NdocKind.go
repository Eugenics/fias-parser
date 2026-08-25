package domain

import "encoding/xml"

type NdocKind struct {
	ID   int    `xml:"ID,attr"`
	Name string `xml:"NAME,attr"`
}

func (o *NdocKind) Decode(decoder *xml.Decoder, se *xml.StartElement) (interface{}, error) {
	err := decoder.DecodeElement(o, se)
	if err != nil {
		return nil, err
	}
	return o, nil
}
