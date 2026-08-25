package domain

import "encoding/xml"

type NdocType struct {
	ID        int    `xml:"ID,attr"`
	Name      string `xml:"NAME,attr"`
	StartDate string `xml:"STARTDATE,attr"`
	EndDate   string `xml:"ENDDATE,attr"`
}

func (o *NdocType) Decode(decoder *xml.Decoder, se *xml.StartElement) (interface{}, error) {
	err := decoder.DecodeElement(o, se)
	if err != nil {
		return nil, err
	}
	return o, nil
}
