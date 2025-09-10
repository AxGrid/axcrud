// transport/response_dto.go
package transport

type ListResponseDTO[DTO any] struct {
	Data  []DTO `json:"data"`
	Total int64 `json:"total"`
}

type OneResponseDTO[DTO any] struct {
	Data DTO `json:"data"`
}

type ManyResponseDTO[DTO any] struct {
	Data []DTO `json:"data"`
}
