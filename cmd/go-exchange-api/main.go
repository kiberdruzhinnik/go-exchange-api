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

	loggingDisabled := os.Getenv("GIN_MODE") == "release"
	if loggingDisabled {
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
	return utils.StringAllowlist(out)
}

func getBaseTicker(c *gin.Context, apiGetTicker func(string) (api.HistoryEntries, error)) {
	ticker := SanitizedParam(c, "ticker")
	data, err := apiGetTicker(ticker)
	if err != nil {
		if err == custom_errors.ErrorNotFound {
			log.Println(err)
			c.JSON(http.StatusNotFound, gin.H{
				"status": "not found",
			})
			return
		} else {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "bad request",
			})
		}
		return
	}
	c.JSON(http.StatusOK, data)
}

func moexGetTicker(c *gin.Context) {
	getBaseTicker(c, MoexAPI.GetTicker)
}

func spbexGetTicker(c *gin.Context) {
	getBaseTicker(c, SpbexAPI.GetTicker)
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
	log.Fatalln(r.Run())
}
