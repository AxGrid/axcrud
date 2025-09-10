package webcrud

import (
	"strings"
)

// ===== Refine inbound =====

type RefineSort struct {
	Field string `json:"field"`
	Order string `json:"order"` // "asc" | "desc"
}

type RefineFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, lt, lte, gt, gte, in, nin, between, contains, icontains, startswith, endswith, isnull
	Value    interface{} `json:"value"`
}

type RefinePagination struct {
	Current  int `json:"current"`  // 1-based
	PageSize int `json:"pageSize"` // per page
}

type RefineListRequest struct {
	Pagination   RefinePagination `json:"pagination"`
	Sorters      []RefineSort     `json:"sorters"`
	Filters      []RefineFilter   `json:"filters"`
	Search       string           `json:"search"`       // опционально
	SearchFields []string         `json:"searchFields"` // опционально
	// Алиас: нередко на фронте зовут поле просто "q"
	Q string `json:"q"`
}

// ===== Repo inbound (из твоего репо-пакета) =====

type Filter struct {
	Field    string
	Operator string
	Value    any
}

type Sort struct {
	Field string
	Order string // "asc"|"desc"
}

type Pagination struct {
	Page    int
	PerPage int
}

type ListParams struct {
	Filters      []Filter
	Sort         *Sort
	Search       string
	SearchFields []string
	Pagination   Pagination
}

// ===== Adapter =====

func AdaptRefineList(req RefineListRequest) ListParams {
	lp := ListParams{
		Pagination: Pagination{
			Page:    max1(req.Pagination.Current),
			PerPage: max1(req.Pagination.PageSize),
		},
	}

	// sorters: берём первый (refine обычно шлёт массив; если хочешь — поддержи multi-sort)
	if len(req.Sorters) > 0 {
		lp.Sort = &Sort{
			Field: req.Sorters[0].Field,
			Order: normalizeOrder(req.Sorters[0].Order),
		}
	}

	// filters
	if len(req.Filters) > 0 {
		lp.Filters = make([]Filter, 0, len(req.Filters))
		for _, f := range req.Filters {
			lp.Filters = append(lp.Filters, Filter{
				Field:    f.Field,
				Operator: strings.ToLower(f.Operator),
				Value:    f.Value,
			})
		}
	}

	// search / q
	search := strings.TrimSpace(req.Search)
	if search == "" {
		search = strings.TrimSpace(req.Q)
	}
	lp.Search = search

	// searchFields
	if len(req.SearchFields) > 0 {
		lp.SearchFields = append(lp.SearchFields, req.SearchFields...)
	}

	return lp
}

func normalizeOrder(s string) string {
	if strings.EqualFold(s, "desc") {
		return "desc"
	}
	return "asc"
}

func max1(v int) int {
	if v <= 0 {
		return 1
	}
	return v
}
