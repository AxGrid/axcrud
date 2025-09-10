package webcrud

import (
	"github.com/axgrid/axcrud"
	"github.com/gin-gonic/gin"
)

func CreateGinRouter[T any, ID IDConstraint](r *gin.RouterGroup, repo axcrud.Repo[T, ID]) {
	r.GET("/", GinGetList[T, ID](repo))
	r.POST("/list", GinPostList[T, ID](repo))
	r.POST("/", GinCreate[T, ID](repo))
	r.GET("/:id", GinGetOne[T, ID](repo))
	r.GET("/many", GinGetMany[T, ID](repo))     // GET ids[]=...
	r.POST("/getMany", GinGetMany[T, ID](repo)) // POST {ids:[]}
	r.PATCH("/:id", GinUpdate[T, ID](repo))
	r.DELETE("/:id", GinDelete[T, ID](repo))
	r.POST("/deleteMany", GinDeleteMany[T, ID](repo))
}
