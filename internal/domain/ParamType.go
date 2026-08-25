package domain

import "encoding/xml"

type ParamType struct {
	ID         int    `xml:"ID,attr"`
	Name       string `xml:"NAME,attr"`
	Desc       string `xml:"DESC,attr"`
	Code       string `xml:"CODE,attr"`
	IsActive   bool   `xml:"ISACTIVE,attr"`
	UpdateDate string `xml:"UPDATEDATE,attr"`
	StartDate  string `xml:"STARTDATE,attr"`
	EndDate    string `xml:"ENDDATE,attr"`
}

func (o *ParamType) Decode(decoder *xml.Decoder, se *xml.StartElement) (interface{}, error) {
	err := decoder.DecodeElement(o, se)
	if err != nil {
		return nil, err
	}
	return o, nil
}
