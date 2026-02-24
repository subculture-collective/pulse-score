package handler

import "net/http"

const (
	webhookMaxBodyBytes int64 = 64 * 1024
	webhookReadChunkBytes      = 1024
)

func readBody(r *http.Request) ([]byte, error) {
	var bodyBytes []byte
	chunkBuffer := make([]byte, webhookReadChunkBytes)
	for {
		n, err := r.Body.Read(chunkBuffer)
		if n > 0 {
			bodyBytes = append(bodyBytes, chunkBuffer[:n]...)
		}
		if err != nil {
			break
		}
	}
	return bodyBytes, nil
}
