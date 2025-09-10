// transport/gin_handlers.go
package webcrud

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Repo[T any, ID IDConstraint] interface {
	GetList(context.Context, ListParams) ([]T, int64, error)
	GetOne(context.Context, ID) (T, error)
	GetMany(context.Context, []ID) ([]T, error)
	Create(context.Context, *T) error
	Update(context.Context, ID, map[string]any) (T, error)
	Delete(context.Context, ID) error
	DeleteMany(context.Context, []ID) (int64, error)
}

func GinGetList[T any, ID IDConstraint](r Repo[T, ID]) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := ParseRefineQuery(c.Request.URL.Query())
		lp := AdaptRefineList(req)
		items, total, err := r.GetList(c, lp)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, ListResponse[T]{Data: items, Total: total})
	}
}

func GinPostList[T any, ID IDConstraint](r Repo[T, ID]) gin.HandlerFunc {
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
		c.JSON(http.StatusOK, ListResponse[T]{Data: items, Total: total})
	}
}

func GinCreate[T any, ID IDConstraint](r Repo[T, ID]) gin.HandlerFunc {
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
		c.JSON(http.StatusOK, OneResponse[T]{Data: in})
	}
}

func GinGetOne[T any, ID IDConstraint](r Repo[T, ID]) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := parseID[ID](idStr)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		item, err := r.GetOne(c, id)
		if err != nil {
			c.AbortWithError(http.StatusNotFound, err)
			return
		}
		c.JSON(http.StatusOK, OneResponse[T]{Data: item})
	}
}

func GinGetMany[T any, ID IDConstraint](r Repo[T, ID]) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in idsReq[ID]
		if err := c.ShouldBindJSON(&in); err != nil {
			// поддержим и GET с query: ids[]=1&ids[]=2
			if c.Request.Method == http.MethodGet {
				idsQ := c.QueryArray("ids[]")
				if len(idsQ) == 0 {
					idsQ = c.QueryArray("ids")
				}
				if len(idsQ) == 0 {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ids required"})
					return
				}
				in.IDs = make([]ID, 0, len(idsQ))
				for _, s := range idsQ {
					id, err := parseID[ID](s)
					if err != nil {
						c.AbortWithError(http.StatusBadRequest, err)
						return
					}
					in.IDs = append(in.IDs, id)
				}
			} else {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		}
		items, err := r.GetMany(c, in.IDs)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, struct {
			Data []T `json:"data"`
		}{Data: items})
	}
}

func GinUpdate[T any, ID IDConstraint](r Repo[T, ID]) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := parseID[ID](idStr)
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
		c.JSON(http.StatusOK, OneResponse[T]{Data: item})
	}
}

func GinDelete[T any, ID IDConstraint](r Repo[T, ID]) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := parseID[ID](idStr)
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

func GinDeleteMany[T any, ID IDConstraint](r Repo[T, ID]) gin.HandlerFunc {
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
