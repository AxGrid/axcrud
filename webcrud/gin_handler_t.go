// transport/gin_handlers_transform.go
package webcrud

import (
	"net/http"

	"github.com/axgrid/axcrud"
	"github.com/gin-gonic/gin"
)

// GET /resource?...
func GinGetListT[T any, ID IDConstraint, DTO any](r axcrud.Repo[T, ID], tr TransformFn[T, DTO]) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := ParseRefineQuery(c.Request.URL.Query())
		lp := AdaptRefineList(req)

		items, total, err := r.GetList(c, lp)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		dtos, err := MapSlice(c, items, tr)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, ListResponseDTO[DTO]{Data: dtos, Total: total})
	}
}

// POST /resource/list  (JSON refine-запрос)
func GinPostListT[T any, ID IDConstraint, DTO any](r axcrud.Repo[T, ID], tr TransformFn[T, DTO]) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in RefineListRequest
		if err := c.ShouldBindJSON(&in); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		lp := AdaptRefineList(in)

		items, total, err := r.GetList(c, lp)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		dtos, err := MapSlice(c, items, tr)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, ListResponseDTO[DTO]{Data: dtos, Total: total})
	}
}

// POST /resource  (create)
func GinCreateT[T any, ID IDConstraint, DTO any](r axcrud.Repo[T, ID], tr TransformFn[T, DTO]) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in T
		if err := c.ShouldBindJSON(&in); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		if err := r.Create(c, &in); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		dto, err := tr(c, in)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, OneResponseDTO[DTO]{Data: dto})
	}
}

// GET /resource/:id (one)
func GinGetOneT[T any, ID IDConstraint, DTO any](r axcrud.Repo[T, ID], tr TransformFn[T, DTO]) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseID[ID](c.Param("id"))
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		item, err := r.GetOne(c, id)
		if err != nil {
			c.AbortWithError(http.StatusNotFound, err)
			return
		}
		dto, err := tr(c, item)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, OneResponseDTO[DTO]{Data: dto})
	}
}

// POST /resource/getMany  { "ids": [...] }  или GET /resource/many?ids[]=...
func GinGetManyT[T any, ID IDConstraint, DTO any](r axcrud.Repo[T, ID], tr TransformFn[T, DTO]) gin.HandlerFunc {
	return func(c *gin.Context) {
		ids, ok, err := readIDsFromRequest[ID](c)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ids required"})
			return
		}

		items, err := r.GetMany(c, ids)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		dtos, err := MapSlice(c, items, tr)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, ManyResponseDTO[DTO]{Data: dtos})
	}
}

// PATCH /resource/:id  (patch -> reload -> transform)
func GinUpdateT[T any, ID IDConstraint, DTO any](r axcrud.Repo[T, ID], tr TransformFn[T, DTO]) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseID[ID](c.Param("id"))
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		var patch map[string]any
		if err := c.ShouldBindJSON(&patch); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		item, err := r.Update(c, id, patch)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		dto, err := tr(c, item)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, OneResponseDTO[DTO]{Data: dto})
	}
}

// DELETE /resource/:id
func GinDeleteT[T any, ID IDConstraint, DTO any](r axcrud.Repo[T, ID]) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseID[ID](c.Param("id"))
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		if err := r.Delete(c, id); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, AffectedResponse{Data: 1})
	}
}

// POST /resource/deleteMany {ids:[]}
func GinDeleteManyT[T any, ID IDConstraint, DTO any](r axcrud.Repo[T, ID]) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in idsReq[ID]
		if err := c.ShouldBindJSON(&in); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		affected, err := r.DeleteMany(c, in.IDs)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, AffectedResponse{Data: affected})
	}
}

// --- утилита чтения ids для GET/POST
func readIDsFromRequest[ID IDConstraint](c *gin.Context) ([]ID, bool, error) {
	// POST JSON
	var in idsReq[ID]
	if err := c.ShouldBindJSON(&in); err == nil && len(in.IDs) > 0 {
		return in.IDs, true, nil
	}
	// GET query
	idsQ := c.QueryArray("ids[]")
	if len(idsQ) == 0 {
		idsQ = c.QueryArray("ids")
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
