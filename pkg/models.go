package pkg

import "google.golang.org/protobuf/compiler/protogen"

type Server struct {
	Service *protogen.Service
	Paths   []APIPath
}

type APIPath struct {
	Method          *protogen.Method
	Tags            []string
	Description     string
	Summary         string
	Path            string
	HTTPMethod      string
	PathParameters  []Parameter
	QueryParameters []Parameter
}

type Parameter struct {
	ModelParameter string
	Key            string
	Type           string
	// resolve Pointer to Input
}
