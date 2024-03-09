package handler

import (
    "server/web/view/index"
    "github.com/labstack/echo/v4"
)

type IndexHandler struct {

}

func (i *IndexHandler) HandleIndexShow(c echo.Context) error {
    return render(c, index.Index("Home", false, "jmisbach"))
}
