package domain

type VersionInfo struct {
	VersionId          int    `json:"VersionId"`
	TextVersion        string `json:"TextVersion"`
	FiasCompleteDbfUrl string `json:"FiasCompleteDbfUrl"`
	FiasCompleteXmlUrl string `json:"FiasCompleteXmlUrl"`
	FiasDeltaDbfUrl    string `json:"FiasDeltaDbfUrl"`
	FiasDeltaXmlUrl    string `json:"FiasDeltaXmlUrl"`
	Kladr4ArjUrl       string `json:"Kladr4ArjUrl"`
	Kladr47ZUrl        string `json:"Kladr47ZUrl"`
	GarXMLFullURL      string `json:"GarXMLFullURL"`
	GarXMLDeltaURL     string `json:"GarXMLDeltaURL"`
	ExpDate            string `json:"ExpDate"`
	Date               string `json:"Date"`
	Status             string
}
