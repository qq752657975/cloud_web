package render

import (
	"github.com/ygb616/web/internal/bytesconv"
	"html/template"
	"net/http"
)

type HTML struct {
	Template   *template.Template
	Data       any
	Name       string
	IsTemplate bool
}

type HTMLRender struct {
	Template *template.Template
}

func (h *HTML) Render(w http.ResponseWriter, code int) error {
	h.WriteContentType(w)
	w.WriteHeader(code)
	if h.IsTemplate {
		err := h.Template.ExecuteTemplate(w, h.Name, h.Data)
		return err
	}
	_, err := w.Write(bytesconv.StringToBytes(h.Data.(string)))
	return err
}

func (h *HTML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "text/html; charset=utf-8")
}
