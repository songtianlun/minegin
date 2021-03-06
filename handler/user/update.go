package user

import (
	"github.com/gin-gonic/gin"
	"mingin/handler"
	"mingin/model"
	"mingin/pkg/errno"
	"mingin/utils"
)

func PutUpdateUser(c *gin.Context) {
	var u model.UserModel
	uid, err := utils.GetUint64ByContext(c, "id")
	if err != nil {
		handler.SendResponse(c, errno.ErrUrl, nil)
	}

	if err := c.Bind(&u); err != nil {
		handler.SendResponse(c, errno.ErrBind, nil)
		return
	}
	nu, err := model.UpdateUserByID(uid, &u)
	if err != nil {
		handler.SendResponse(c, errno.ErrUpdatePref, nil)
		return
	}
	handler.SendResponse(c, nil, nu)
}
