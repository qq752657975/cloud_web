package render

import (
	"errors"
	"fmt"
	"net/http"
)

type Redirect struct {
	Code     int
	Request  *http.Request
	Location string
}

func (r Redirect) Render(w http.ResponseWriter, code int) error {
	//如果小于300 大于308 并且不等于 201 则抛出异常
	if (r.Code < http.StatusMultipleChoices || r.Code > http.StatusPermanentRedirect) && r.Code != http.StatusCreated {
		return errors.New(fmt.Sprintf("Cannot redirect with status code %d", r.Code))
	}
	http.Redirect(w, r.Request, r.Location, r.Code)
	return nil
}

// WriteContentType (Redirect) don't write any ContentType.
func (r Redirect) WriteContentType(http.ResponseWriter) {

}
