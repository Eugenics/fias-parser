package domain

import "encoding/xml"

type ObjectLevel struct {
	Level      int    `xml:"LEVEL,attr"`
	Name       string `xml:"NAME,attr"`
	StartDate  string `xml:"STARTDATE,attr"`
	EndDate    string `xml:"ENDDATE,attr"`
	UpdateDate string `xml:"UPDATEDATE,attr"`
	IsActive   bool   `xml:"ISACTIVE,attr"`
}

func (o *ObjectLevel) Decode(decoder *xml.Decoder, se *xml.StartElement) (interface{}, error) {
	err := decoder.DecodeElement(o, se)
	if err != nil {
		return nil, err
	}
	return o, nil
}
