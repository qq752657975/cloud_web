package render

import (
	"fmt"
	"github.com/ygb616/web/internal/bytesconv"
	"net/http"
)

type String struct {
	Format string
	Data   []any
}

func (s *String) Render(w http.ResponseWriter, code int) error {
	s.WriteContentType(w)
	w.WriteHeader(code)
	if len(s.Data) > 0 {
		_, err := fmt.Fprintf(w, s.Format, s.Data...)
		return err
	}
	_, err := w.Write(bytesconv.StringToBytes(s.Format))
	return err
}

func (s *String) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "text/plan; charset=utf-8")
}
