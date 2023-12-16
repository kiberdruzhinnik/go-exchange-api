package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kiberdruzhinnik/go-exchange-api/api"
	custom_errors "github.com/kiberdruzhinnik/go-exchange-api/errors"
	"github.com/kiberdruzhinnik/go-exchange-api/utils"
)

var MoexAPI api.MoexAPI
var SpbexAPI api.SpbexAPI

func init() {
	var redisClient utils.RedisClient
	redisUrl := os.Getenv("EXCHANGE_API_REDIS")
	if redisUrl != "" {
		log.Println("Got redis connection string, trying to create connection")
		client, err := utils.NewRedisClient(redisUrl)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("Redis is connected")
		redisClient = client
	}

	loggingEnabled := os.Getenv("EXCHANGE_API_VERBOSE")
	if len(loggingEnabled) == 0 {
		log.SetOutput(io.Discard)
	} else {
		log.Println("Verbose logging enabled")
	}

	MoexAPI = api.NewMoexAPI(redisClient)
	SpbexAPI = api.NewSpbexAPI()
}

func SanitizedParam(c *gin.Context, param string) string {
	out := c.Param(param)
	out = strings.ToLower(out)
	return utils.RemoveNonAlnum(out)
}

func moexGetTicker(c *gin.Context) {
	ticker := SanitizedParam(c, "ticker")
	data, err := MoexAPI.GetTicker(ticker)
	if err != nil {
		if err == custom_errors.ErrorNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"status": "not found",
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "bad request",
			})
		}
		return
	}

	c.JSON(http.StatusOK, data)
}

func spbexGetTicker(c *gin.Context) {
	ticker := SanitizedParam(c, "ticker")
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data":   ticker,
	})
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func mountRoutes(app *gin.Engine) {
	app.GET("/moex/:ticker", moexGetTicker)
	app.GET("/spbex/:ticker", spbexGetTicker)
	app.GET("/healthcheck", healthCheck)
}

func main() {
	r := gin.Default()
	mountRoutes(r)
	r.Run()
}
