package domain

import "encoding/xml"

type Param struct {
	Id          int    `xml:"ID,attr"`
	ObjectId    string `xml:"OBJECTID,attr"`
	ChangeId    string `xml:"CHANGEID,attr"`
	ChangeIdEnd string `xml:"CHANGEIDEND,attr"`
	TypeId      string `xml:"TYPEID,attr"`
	Value       string `xml:"VALUE,attr"`
	UpdateDate  string `xml:"UPDATEDATE,attr"`
	StartDate   string `xml:"STARTDATE,attr"`
	EndDate     string `xml:"ENDDATE,attr"`
}

func (o *Param) Decode(decoder *xml.Decoder, se *xml.StartElement) (interface{}, error) {
	err := decoder.DecodeElement(o, se)
	if err != nil {
		return nil, err
	}
	return o, nil
}
