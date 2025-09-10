package webcrud

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// TransformFn — функция преобразования доменной модели в DTO для выдачи наружу.
type TransformFn[T any, DTO any] func(ctx context.Context, src T) (DTO, error)

// MapSlice — утилита для маппинга слайса через TransformFn.
func MapSlice[T any, DTO any](ctx context.Context, in []T, fn TransformFn[T, DTO]) ([]DTO, error) {
	if fn == nil {
		// identity по типам — не получится, поэтому лучше падать явно
		return nil, fmt.Errorf("transform fn is nil")
	}
	out := make([]DTO, len(in))
	for i := range in {
		dto, err := fn(ctx, in[i])
		if err != nil {
			return nil, err
		}
		out[i] = dto
	}
	return out, nil
}

/*
NewIDHasher — фабрика HMAC-SHA256 → base64url (обрезанный), чтобы заменять порядковый ID на стабильный хеш.
- salt: любой секрет/pepper (храните в конфиге)
- n: длина base64url-строки (например, 11–16)

Пример:

	hasher := NewIDHasher("mysalt", 11)
	hash := hasher(fmt.Sprint(123)) // "Qm1c2eC1yVZ"
*/
func NewIDHasher(salt string, n int) func(idString string) string {
	return func(idString string) string {
		m := hmac.New(sha256.New, []byte(salt))
		m.Write([]byte(idString))
		sum := m.Sum(nil)
		s := base64.RawURLEncoding.EncodeToString(sum)
		if n > 0 && n < len(s) {
			return s[:n]
		}
		return s
	}
}
