// transport/refine_query.go
package transport

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func ParseRefineQuery(values url.Values) RefineListRequest {
	var req RefineListRequest

	// pagination
	req.Pagination.Current = atoi(values.Get("current"))
	req.Pagination.PageSize = atoi(values.Get("pageSize"))

	// search / q
	req.Search = values.Get("search")
	if req.Search == "" {
		req.Q = values.Get("q")
	}

	// searchFields[]
	req.SearchFields = values["searchFields[]"]
	if len(req.SearchFields) == 0 {
		req.SearchFields = values["searchFields"] // иногда без []
	}

	// sorters: полноценный массив sorters[i][field]
	sortIdx := collectIndexed(values, "sorters")
	for _, i := range sortIdx {
		field := values.Get(key2("sorters", i, "field"))
		order := values.Get(key2("sorters", i, "order"))
		if field != "" {
			req.Sorters = append(req.Sorters, RefineSort{Field: field, Order: order})
		}
	}
	// упрощённая форма: ?sorters=field&order=asc
	if len(req.Sorters) == 0 {
		if f := values.Get("sorters"); f != "" {
			req.Sorters = append(req.Sorters, RefineSort{
				Field: f, Order: values.Get("order"),
			})
		}
	}

	// filters: filters[i][field], [operator], [value] или [value][]
	fIdx := collectIndexed(values, "filters")
	for _, i := range fIdx {
		f := RefineFilter{
			Field:    values.Get(key2("filters", i, "field")),
			Operator: values.Get(key2("filters", i, "operator")),
		}
		// value может быть скаляром или массивом
		arr := values[key2("filters", i, "value[]")]
		if len(arr) == 0 {
			arr = values[key2("filters", i, "value")] // если бек прислал как многократный
		}
		if len(arr) > 0 {
			vs := make([]any, 0, len(arr))
			for _, v := range arr {
				vs = append(vs, v)
			}
			f.Value = vs
		} else {
			f.Value = values.Get(key2("filters", i, "value"))
		}
		if f.Field != "" {
			req.Filters = append(req.Filters, f)
		}
	}

	return req
}

// ==== helpers (локальные) ====

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func key2(root string, idx int, leaf string) string {
	return fmt.Sprintf("%s[%d][%s]", root, idx, leaf)
}

func collectIndexed(v url.Values, root string) []int {
	// ищем ключи вида root[<n>][...]
	seen := map[int]struct{}{}
	for k := range v {
		if !strings.HasPrefix(k, root+"[") {
			continue
		}
		// выдёргиваем число между [ ]
		open := strings.Index(k, "[")
		closed := strings.Index(k[open+1:], "]")
		if open >= 0 && closed > 0 {
			idxStr := k[open+1 : open+1+closed]
			if i, err := strconv.Atoi(idxStr); err == nil {
				seen[i] = struct{}{}
			}
		}
	}
	out := make([]int, 0, len(seen))
	for i := range seen {
		out = append(out, i)
	}
	sort.Ints(out)
	return out
}
