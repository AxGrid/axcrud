package webcrud

type ListResponse[T any] struct {
	Data  []T   `json:"data"`
	Total int64 `json:"total"`
}

type OneResponse[T any] struct {
	Data T `json:"data"`
}

type AffectedResponse struct {
	Data int64 `json:"data"`
}
