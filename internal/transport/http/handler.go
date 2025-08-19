package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	_ "wb-l0-go/docs"

	gin "github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"wb-l0-go/internal/domain"
	"wb-l0-go/internal/service"
	kafkaTransport "wb-l0-go/internal/transport/kafka"
)

type Handler struct {
	service  *service.OrderService
	log      *zap.Logger
	producer *kafkaTransport.Producer
}

func NewHandler(svc *service.OrderService, prod *kafkaTransport.Producer, log *zap.Logger) *Handler {
	return &Handler{service: svc, producer: prod, log: log}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.Static("/static", "./internal/frontend")
	r.GET("/", func(c *gin.Context) {
		c.File("./internal/frontend/index.html")
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/orders", h.listOrders)
	r.GET("/orders/:order_uid", h.getOrder)
	r.POST("/publish", h.publish)
}

// @Summary      Список uid заказов
// @Description  Получить список uid заказов
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        limit  query    int  false  "Limit"
// @Param        offset query    int  false  "Offset"
// @Success      200  {array}  string
// @Failure      500  {object}  map[string]interface{}
// @Router       /orders [get]
func (h *Handler) listOrders(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	orders, err := h.service.ListOrdersUIDs(c.Request.Context(), limit, offset)
	if err != nil {
		h.log.Error("failed to list orders", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, orders)
}

// @Summary      Получить заказ по uid
// @Description  Получить заказ по uid
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        order_uid  path    string  true  "Order UID"
// @Success      200  {object}  domain.Order
// @Failure      500  {object}  map[string]interface{}
// @Router       /orders/{order_uid} [get]
func (h *Handler) getOrder(c *gin.Context) {
	orderUID := c.Param("order_uid")
	order, err := h.service.GetOrder(c.Request.Context(), orderUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		h.log.Error("failed to get order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, order)
}

// @Summary      Опубликовать заказ
// @Description  Опубликовать заказ в Kafka
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        order body domain.Order true "Order"
// @Success      202  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /publish [post]
func (h *Handler) publish(c *gin.Context) {
	var order domain.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	if order.OrderUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_uid is required"})
		return
	}
	payload, err := json.Marshal(order)
	if err != nil {
		h.log.Error("failed to marshal order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if h.producer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "producer not initialized"})
		return
	}
	if err := h.producer.Publish(c.Request.Context(), order.OrderUID, payload); err != nil {
		h.log.Error("failed to publish", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to publish"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "published", "order_uid": order.OrderUID})
}
