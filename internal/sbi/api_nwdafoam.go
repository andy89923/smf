package sbi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/nwdaf/pkg/components"
	nwdafModels "github.com/free5gc/nwdaf/pkg/models"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/logger"

	smfcontext "github.com/free5gc/smf/internal/context"
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
			Name:    "SMF NF_Load Resouce",
			Method:  http.MethodGet,
			Pattern: "/nf-load",
			APIFunc: s.SmfNfLoadOamGet,
		},
		{
			Name:    "NfLoadLevelAnalyticsNotification",
			Method:  http.MethodPost,
			Pattern: "/callback",
			APIFunc: s.NfLoadLevelAnalyticsNotification,
		},
	}
}

func (s *Server) SmfNfLoadOamGet(c *gin.Context) {
	ues := s.Context().GetUesData()
	totalPduCount := ues.GetTotalPduSessionCount()

	smContextPool := smfcontext.GetSmContextPool()
	poolSize := 0
	smContextPool.Range(
		func(key, value any) bool {
			poolSize++
			return true // return false to stop iterating early
		})

	urrThreshold := s.Processor().ChargingUrrThreshold

	nfResource, err := components.GetNfResouces(context.Background())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	smfNfLoadOam := nwdafModels.SmfNfLoadOamResponse{
		NfResource:    *nfResource,
		UrrThresholds: urrThreshold,
		NumSmContext:  uint64(poolSize),
		NumPduSession: totalPduCount,
	}

	// logger.SBILog.Warnf("SmfNfLoadOamResponse: %+v", smfNfLoadOam)

	c.JSON(http.StatusOK, smfNfLoadOam)

	go s.Processor().SubscribeNfLoadIfNotExist(s.CancelContext())
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
