package main

type Method struct {
	// Name of the method
	Name string `json:"name"`
	// Endpoint of the method
	Endpoint string `json:"endpoint"`
}

var getBlockInfo = Method{
	Name:     "getBlockInfo",
	Endpoint: "blockchain",
}

func (m Method) IsMethod(method string) bool {
	return m.Name == method || m.Endpoint+"."+m.Name == method
}
