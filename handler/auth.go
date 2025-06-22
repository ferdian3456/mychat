package handler

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func (h *Handler) HelloPrint(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	fmt.Fprintf(writer, "wow")
}
