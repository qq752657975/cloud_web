package render

import (
	"encoding/xml"
	"net/http"
)

type XML struct {
	Data any
}

func (x *XML) Render(w http.ResponseWriter, code int) error {
	x.WriteContentType(w)
	w.WriteHeader(code)
	return xml.NewEncoder(w).Encode(x.Data)
}

func (x *XML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "application/xml; charset=utf-8")
}
