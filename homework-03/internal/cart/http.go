package cart

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) http.Handler {
	handler := &HTTPHandler{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /user/{user_id}/cart/{sku_id}", handler.add)
	mux.HandleFunc("GET /user/{user_id}/cart", handler.get)
	mux.HandleFunc("GET /swagger.json", handler.swagger)
	return logging(mux)
}

type addRequest struct {
	Count uint32 `json:"count"`
}

type itemResponse struct {
	SKU   int64  `json:"sku"`
	Name  string `json:"name"`
	Count uint32 `json:"count"`
	Price uint32 `json:"price"`
}

type cartResponse struct {
	Items      []itemResponse `json:"items"`
	TotalPrice uint32         `json:"total_price"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *HTTPHandler) add(writer http.ResponseWriter, request *http.Request) {
	userID, err := positiveID(request.PathValue("user_id"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "user_id must be a positive integer")
		return
	}
	sku, err := positiveID(request.PathValue("sku_id"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "sku_id must be a positive integer")
		return
	}

	request.Body = http.MaxBytesReader(writer, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var body addRequest
	if err := decoder.Decode(&body); err != nil {
		writeError(writer, http.StatusBadRequest, "body must be JSON with positive count")
		return
	}
	if err := ensureJSONEOF(decoder); err != nil || body.Count == 0 {
		writeError(writer, http.StatusBadRequest, "count must be a positive integer")
		return
	}

	if err := h.service.Add(request.Context(), userID, sku, body.Count); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			writeError(writer, http.StatusPreconditionFailed, "product does not exist")
			return
		}
		writeError(writer, http.StatusInternalServerError, "cannot add product to cart")
		return
	}
	writer.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) get(writer http.ResponseWriter, request *http.Request) {
	userID, err := positiveID(request.PathValue("user_id"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "user_id must be a positive integer")
		return
	}

	cart, err := h.service.Get(request.Context(), userID)
	if errors.Is(err, ErrCartNotFound) {
		writeError(writer, http.StatusNotFound, "cart not found")
		return
	}
	if errors.Is(err, ErrProductNotFound) {
		writeError(writer, http.StatusPreconditionFailed, "product does not exist")
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "cannot get cart")
		return
	}

	response := cartResponse{Items: make([]itemResponse, 0, len(cart.Items)), TotalPrice: cart.TotalPrice}
	for _, item := range cart.Items {
		response.Items = append(response.Items, itemResponse{
			SKU: item.SKU, Name: item.Name, Count: item.Count, Price: item.Price,
		})
	}
	writeJSON(writer, http.StatusOK, response)
}

func (h *HTTPHandler) swagger(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = writer.Write(swaggerJSON)
}

func positiveID(value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, errors.New("not a positive integer")
	}
	return parsed, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, errorResponse{Error: message})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		log.Printf("encode HTTP response: %v", err)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(wrapped, request)
		route := request.Pattern
		if route == "" {
			route = "unmatched"
		}
		log.Printf("method=%s route=%q status=%d duration=%s", request.Method, route, wrapped.status, time.Since(started))
	})
}
