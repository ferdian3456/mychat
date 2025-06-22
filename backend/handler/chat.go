package handler

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func (h *Handler) SendMessage(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	fmt.Fprintf(writer, "woiii")
}

func (h *Handler) GetMessages(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {

}
