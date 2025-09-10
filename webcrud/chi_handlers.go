package webcrud

import (
	"encoding/json"
	"net/http"

	"github.com/axgrid/axcrud"
	"github.com/go-chi/chi/v5"
)

func ChiGetList[T any, ID IDConstraint](r axcrud.Repo[T, ID]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		rreq := ParseRefineQuery(req.URL.Query())
		lp := AdaptRefineList(rreq)
		items, total, err := r.GetList(req.Context(), lp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		WriteJSON(w, http.StatusOK, ListResponse[T]{Data: items, Total: total})
	}
}

func ChiPostList[T any, ID IDConstraint](r axcrud.Repo[T, ID]) http.HandlerFunc {
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
		WriteJSON(w, http.StatusOK, ListResponse[T]{Data: items, Total: total})
	}
}

func ChiCreate[T any, ID IDConstraint](r axcrud.Repo[T, ID]) http.HandlerFunc {
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
		WriteJSON(w, http.StatusOK, OneResponse[T]{Data: in})
	}
}

func ChiGetOne[T any, ID IDConstraint](r axcrud.Repo[T, ID]) http.HandlerFunc {
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
		WriteJSON(w, http.StatusOK, OneResponse[T]{Data: item})
	}
}

func ChiGetMany[T any, ID IDConstraint](r axcrud.Repo[T, ID]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			idsQ := req.URL.Query()["ids[]"]
			if len(idsQ) == 0 {
				idsQ = req.URL.Query()["ids"]
			}
			if len(idsQ) == 0 {
				http.Error(w, "ids required", http.StatusBadRequest)
				return
			}
			ids := make([]ID, 0, len(idsQ))
			for _, s := range idsQ {
				id, err := parseID[ID](s)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				ids = append(ids, id)
			}
			items, err := r.GetMany(req.Context(), ids)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			WriteJSON(w, http.StatusOK, struct {
				Data []T `json:"data"`
			}{Data: items})
			return
		}
		var in idsReq[ID]
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		items, err := r.GetMany(req.Context(), in.IDs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		WriteJSON(w, http.StatusOK, struct {
			Data []T `json:"data"`
		}{Data: items})
	}
}

func ChiUpdate[T any, ID ~uint | ~uint64 | ~int | ~int64 | ~string](r axcrud.Repo[T, ID]) http.HandlerFunc {
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
		WriteJSON(w, http.StatusOK, OneResponse[T]{Data: item})
	}
}

func ChiDelete[T any, ID ~uint | ~uint64 | ~int | ~int64 | ~string](r axcrud.Repo[T, ID]) http.HandlerFunc {
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

func ChiDeleteMany[T any, ID IDConstraint](r axcrud.Repo[T, ID]) http.HandlerFunc {
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
