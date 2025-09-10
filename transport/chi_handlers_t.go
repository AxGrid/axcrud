package transport

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GET /resource?current=&pageSize=&sorters[...]&filters[...]&q=...
func ChiGetListT[T any, ID IDConstraint, DTO any](r Repo[T, ID], tr TransformFn[T, DTO]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ref := ParseRefineQuery(req.URL.Query())
		lp := AdaptRefineList(ref)

		items, total, err := r.GetList(req.Context(), lp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		dtos, err := MapSlice(req.Context(), items, tr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		WriteJSON(w, http.StatusOK, ListResponseDTO[DTO]{Data: dtos, Total: total})
	}
}

// POST /resource/list  (JSON {pagination, sorters, filters, search/searchFields/q})
func ChiPostListT[T any, ID IDConstraint, DTO any](r Repo[T, ID], tr TransformFn[T, DTO]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var in RefineListRequest
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lp := AdaptRefineList(in)

		items, total, err := r.GetList(req.Context(), lp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		dtos, err := MapSlice(req.Context(), items, tr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		WriteJSON(w, http.StatusOK, ListResponseDTO[DTO]{Data: dtos, Total: total})
	}
}

// POST /resource  (create) — тело: доменная модель T
func ChiCreateT[T any, ID IDConstraint, DTO any](r Repo[T, ID], tr TransformFn[T, DTO]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var in T
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := r.Create(req.Context(), &in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		dto, err := tr(req.Context(), in)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		WriteJSON(w, http.StatusOK, OneResponseDTO[DTO]{Data: dto})
	}
}

// GET /resource/{id}
func ChiGetOneT[T any, ID IDConstraint, DTO any](r Repo[T, ID], tr TransformFn[T, DTO]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		idStr := chi.URLParam(req, "id")
		id, err := parseID[ID](idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		item, err := r.GetOne(req.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		dto, err := tr(req.Context(), item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		WriteJSON(w, http.StatusOK, OneResponseDTO[DTO]{Data: dto})
	}
}

// GET /resource/many?ids[]=...  И/ИЛИ  POST /resource/getMany  { "ids": [...] }
func ChiGetManyT[T any, ID IDConstraint, DTO any](r Repo[T, ID], tr TransformFn[T, DTO]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ids, ok, err := readIDsChi[ID](req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, "ids required", http.StatusBadRequest)
			return
		}

		items, err := r.GetMany(req.Context(), ids)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		dtos, err := MapSlice(req.Context(), items, tr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		WriteJSON(w, http.StatusOK, ManyResponseDTO[DTO]{Data: dtos})
	}
}

// PATCH /resource/{id}  (тело: map[string]any)
func ChiUpdateT[T any, ID IDConstraint, DTO any](r Repo[T, ID], tr TransformFn[T, DTO]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		idStr := chi.URLParam(req, "id")
		id, err := parseID[ID](idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var patch map[string]any
		if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		item, err := r.Update(req.Context(), id, patch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		dto, err := tr(req.Context(), item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		WriteJSON(w, http.StatusOK, OneResponseDTO[DTO]{Data: dto})
	}
}

// DELETE /resource/{id}
func ChiDeleteT[T any, ID IDConstraint, DTO any](r Repo[T, ID]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		idStr := chi.URLParam(req, "id")
		id, err := parseID[ID](idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := r.Delete(req.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		WriteJSON(w, http.StatusOK, AffectedResponse{Data: 1})
	}
}

// POST /resource/deleteMany  { "ids": [...] }
func ChiDeleteManyT[T any, ID IDConstraint, DTO any](r Repo[T, ID]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var in idsReq[ID]
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		affected, err := r.DeleteMany(req.Context(), in.IDs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		WriteJSON(w, http.StatusOK, AffectedResponse{Data: affected})
	}
}

// --- утилита чтения ids из GET/POST
func readIDsChi[ID IDConstraint](req *http.Request) ([]ID, bool, error) {
	// Если POST JSON — пробуем прочитать тело (не «съедаем» внешним декодером).
	if req.Method == http.MethodPost {
		var in idsReq[ID]
		if err := json.NewDecoder(req.Body).Decode(&in); err == nil && len(in.IDs) > 0 {
			return in.IDs, true, nil
		}
		// если не получилось — идём дальше, вдруг query
	}

	// GET / ... ?ids[]=1&ids[]=2
	idsQ := req.URL.Query()["ids[]"]
	if len(idsQ) == 0 {
		idsQ = req.URL.Query()["ids"]
	}
	if len(idsQ) == 0 {
		return nil, false, nil
	}

	out := make([]ID, 0, len(idsQ))
	for _, s := range idsQ {
		id, err := parseID[ID](s)
		if err != nil {
			return nil, false, err
		}
		out = append(out, id)
	}
	return out, true, nil
}
