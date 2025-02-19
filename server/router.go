package server

import (
	"strings"
	"sushi/utils"
	"sushi/utils/config"
	"sushi/utils/ratelimit"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
)

func NewRouter(server *Server, conf config.Config, socketserver *socketio.Server) *gin.Engine {
	gin.SetMode(server.config.GinMode())
	r := gin.Default()

	r.Use(ratelimit.GinMiddleware())
	r.Use(CORSMiddleware())
	r.Use(static.Serve("/", static.LocalFile("/app/Demo-UI", true)))

	//public
	r.GET("/ping", server.controller.HandlePing)
	r.POST("/users/exist", server.controller.HandleGetUserExist)

	//from game server(RSA 签名)
	// API v1
	// r.POST("/earn", server.controller.HandleEarn)
	// API v2: support freebie
	r.POST("/earn", server.controller.HandleEarnAllowFreebie)

	//from player

	//Todo
	r.GET("/swap_info") //balance rate swap_limit record(3)
	//r.PUT("/withdraw", server.controller.HandleApplyWithdraw)

	//from tx server
	//r.POST("/withdraw", server.controller.HandleHandleWithdraw)
	//r.PATCH("/withdraw", server.controller.HandleConfirmWithdraw)

	//from admin(JWT 验证Mail,Sub)
	//update rate
	//update swapLimit

	v1 := r.Group("/v1")
	authorizedV1 := v1.Group("/")
	authorizedV1.Use(server.GetAuth())
	//
	//v1Team := v1.Group("/teams")
	//v1Team.Use(server.GetAuth())
	//
	//v2 := r.Group("/v2")
	//v2_authorized := v2.Group("/")
	//v2_authorized.Use(server.GetAuth())

	//r.GET("/socket/*any", gin.WrapH(socketserver))
	//r.POST("/socket/*any", gin.WrapH(socketserver))
	//v1.POST("/user/signup", server.controller.user.HandleSignup)
	//
	//
	WithUserRoutes(authorizedV1, server, conf)
	//
	//
	//WithTeamRoutes(v1Team, server)
	return r
}

func WithTeamRoutes(r *gin.RouterGroup, server *Server) {
	//r.GET("/", server.controller.team.HandleTeamList)
}

func WithUserRoutes(r *gin.RouterGroup, server *Server, conf config.Config) {
	authorized := r
	//authorized.POST("/users/profile")
	authorized.GET("/users/profile", server.controller.HandleGetUserInfo)
	authorized.POST("/player", server.controller.HandleNewPlayer)
	authorized.POST("/swap", server.controller.HandleSwap)
	authorized.GET("/balance", server.controller.HandleGetBalance)
	authorized.GET("/month_withdraw", server.controller.HandleGetMonthWithdraw)
	authorized.GET("/earn_record", server.controller.HandleGetEarnRecords)
	authorized.GET("/swap_record", server.controller.HandleGetSwapRecords)
	authorized.GET("/withdraw_record", server.controller.HandleGetWithdrawRecords)
	authorized.GET("/earn_total", server.controller.HandleGetEarnTotal)
	authorized.GET("/withdraw_total", server.controller.HandleGetWithdrawTotal)
	authorized.POST("/ethaddr", server.controller.HandleEditEthAddress)
	authorized.GET("/nfts", server.controller.HandleGetNfts)
	authorized.GET("/freebie_record", server.controller.HandleGetFreebieRecords)
	//authorized.POST("/users/profile", server.controller.user.HandleUpdateUserInfo)
	//authorized.GET("/users/profile", server.controller.user.HandleGetUserInfo)
	//authorized.POST("/urls", server.controller.preSignURL.HandleURLRegister)
}

func (server Server) GetAuth() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.Set("sub", "")
		token := c.GetHeader("Authorization")

		parts := strings.Split(token, " ")
		if parts[1] == "" {
			utils.ErrorResponse(c, 400, "token not found", "")
			return
		}

		userInfo, err := utils.GetUserInfo(server.config.Auth0URL()+"/userinfo", c, token)
		if err != nil {
			utils.ErrorResponse(c, 500, err.Error(), "")
			return
		}

		c.Set("sub", userInfo.Sub)
		c.Set("name", userInfo.Name)
		c.Set("mail", userInfo.Email)
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, X-Auth-Token, Authorization, Code, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT , PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func adminAuth() gin.HandlerFunc {
	accounts := gin.Accounts{
		"larry":     "larrykey",
		"scheduler": "schedulerkey",
	}
	return gin.BasicAuth(accounts)
}
