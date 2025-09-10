// transport/common.go
package webcrud

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type idsReq[ID any] struct {
	IDs []ID `json:"ids"`
}

type IDConstraint interface {
	~uint | ~uint64 | ~int | ~int64 | ~string
}

func parseID[ID IDConstraint](s string) (ID, error) {
	var id ID
	switch any(id).(type) {
	case string:
		return any(s).(ID), nil
	case int:
		n, err := strconv.Atoi(s)
		if err != nil {
			return id, err
		}
		return any(n).(ID), nil
	case int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return id, err
		}
		return any(n).(ID), nil
	case uint:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return id, err
		}
		return any(uint(n)).(ID), nil
	case uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return id, err
		}
		return any(n).(ID), nil
	default:
		return id, fmt.Errorf("unsupported ID type")
	}
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
