package webcrud

import (
	"github.com/axgrid/axcrud"
	"github.com/gin-gonic/gin"
)

func CreateGinRouter[T any, ID IDConstraint](r *gin.RouterGroup, repo axcrud.Repo[T, ID]) {
	r.GET("/users", GinGetList[T, ID](repo))
	r.POST("/users/list", GinPostList[T, ID](repo))
	r.POST("/users", GinCreate[T, ID](repo))
	r.GET("/users/:id", GinGetOne[T, ID](repo))
	r.GET("/users/many", GinGetMany[T, ID](repo))     // GET ids[]=...
	r.POST("/users/getMany", GinGetMany[T, ID](repo)) // POST {ids:[]}
	r.PATCH("/users/:id", GinUpdate[T, ID](repo))
	r.DELETE("/users/:id", GinDelete[T, ID](repo))
	r.POST("/users/deleteMany", GinDeleteMany[T, ID](repo))
}
