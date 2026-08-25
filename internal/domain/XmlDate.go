package domain

import (
	"encoding/xml"
	"time"
)

type XMLDate struct {
	time.Time
}

func (d *XMLDate) UnmarshalXMLAttr(attr xml.Attr) error {
	t, err := time.Parse("2006-01-02", attr.Value)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}
