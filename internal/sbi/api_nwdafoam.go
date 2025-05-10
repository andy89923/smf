package sbi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/nwdaf/pkg/components"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/logger"
)

func (s *Server) getNwdafOamRoutes() []Route {
	return []Route{
		{
			Name:    "Health Check",
			Method:  http.MethodGet,
			Pattern: "/",
			APIFunc: func(c *gin.Context) {
				c.String(http.StatusOK, "SMF NWDAF-OAM woking!")
			},
		},
		{
			Name:    "NfResourceGet",
			Method:  http.MethodGet,
			Pattern: "/nf-resource",
			APIFunc: s.SmfOamNfResourceGet,
		},
		{
			Name:    "NfLoadLevelAnalyticsNotification",
			Method:  http.MethodPost,
			Pattern: "/callback",
			APIFunc: s.NfLoadLevelAnalyticsNotification,
		},
	}
}

func (s *Server) SmfOamNfResourceGet(c *gin.Context) {
	nfResource, err := components.GetNfResouces(context.Background())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, *nfResource)

	// If this handler function be called, which means NWDAF in running in this core.
	// So to Subscribe to NWDAF for CHF, SMF, and UPF analytics.
	go s.Processor().SubscribeNfLoadIfNotExist(s.CancelContext())
}

func (s *Server) NfLoadLevelAnalyticsNotification(c *gin.Context) {
	logger.SBILog.Infoln("Receive NfLoadLevelAnalyticsNotification")

	var notification []models.NnwdafEventsSubscriptionNotification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "Invalid JSON format",
			Detail: err.Error(),
		})
		return
	}
	if len(notification) == 0 {
		c.JSON(http.StatusBadRequest, models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "Notification is empty",
			Detail: "Empty notification",
		})
		return
	}
	logger.SBILog.Infoln("Notification:", notification)

	s.Processor().ReceiveNfLoadLevelAnalytics(&notification)

	c.Status(http.StatusNoContent)
}
