package controllers

import (
	"net/http"

	"github.com/nrmnqdds/gomaluum-api/dtos"
	"github.com/nrmnqdds/gomaluum-api/internal"
	"github.com/nrmnqdds/gomaluum-api/services/auth"
	"github.com/nrmnqdds/gomaluum-api/services/scraper"

	"github.com/labstack/echo/v4"
)

var logger = internal.NewLogger()

// @Title GetScheduleHandler
// @Description Get schedule from i-Ma'luum
// @Tags scraper
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/schedule [get]
func GetScheduleHandler(c echo.Context) error {
	schedule := dtos.ScheduleRequestProps{
		Echo: c,
	}

	data, err := scraper.ScheduleScraper(&schedule)
	if err != nil {
		response := dtos.Response{
			Status:  err.StatusCode,
			Message: err.Message,
			Data:    nil,
		}
		logger.Error(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	response := dtos.Response{
		Status:  http.StatusOK,
		Message: "Successfully get user schedule!",
		Data:    data,
	}

	return c.JSON(http.StatusOK, response)
}

func PostScheduleHandler(c echo.Context) error {
	user := dtos.LoginDTO{}

	if c.Bind(&user) != nil {

		logger.Error("Invalid request payload!")

		response := dtos.Response{
			Status:  http.StatusBadRequest,
			Message: "Invalid request payload!",
			Data:    nil,
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	if validationErr := internal.Validator.Struct(&user); validationErr != nil {
		response := dtos.Response{
			Status:  http.StatusBadRequest,
			Message: "Invalid request payload!",
			Data:    nil,
		}
		logger.Error("Invalid request payload!", validationErr)
		return c.JSON(http.StatusBadRequest, response)
	}

	// Set cookie to delete the previous cookie
	c.SetCookie(&http.Cookie{
		Name:   "MOD_AUTH_CAS",
		Value:  "test",
		Path:   "/",
		MaxAge: -1,

		HttpOnly: true,
	})

	loginRes, err := auth.LoginUser(&user)
	if err != nil {
		response := dtos.Response{
			Status:  err.StatusCode,
			Message: err.Message,
			Data:    nil,
		}
		logger.Error(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	logger.Infof("Successfully login! Token: %s", loginRes.Token)

	schedule := dtos.ScheduleRequestProps{
		Echo:  c,
		Token: loginRes.Token,
	}

	logger.Infof("Schedule request props: %+v", schedule)

	data, err := scraper.ScheduleScraper(&schedule)
	if err != nil {
		response := dtos.Response{
			Status:  err.StatusCode,
			Message: err.Message,
			Data:    nil,
		}
		logger.Error(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	response := dtos.Response{
		Status:  http.StatusOK,
		Message: "Successfully get user schedule!",
		Data:    data,
	}

	return c.JSON(http.StatusOK, response)
}
