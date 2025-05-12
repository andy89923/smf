package processor

import (
	"context"
	"fmt"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/logger"
)

func (p *Processor) ReceiveNfLoadLevelAnalytics(notification *[]models.NnwdafEventsSubscriptionNotification) {
	if (notification == nil) || (len(*notification) == 0) {
		logger.ProcessorLog.Warnln("ReceiveNfLoadLevelAnalytics: notification is nil or empty")
		return
	}
	if len((*notification)[0].EventNotifications) == 0 {
		logger.ProcessorLog.Warnln("ReceiveNfLoadLevelAnalytics: EventNotifications is nil or empty")
		return
	}
	eventNotification := (*notification)[0].EventNotifications[0]
	if len(eventNotification.NfLoadLevelInfos) == 0 {
		logger.ProcessorLog.Warnln("ReceiveNfLoadLevelAnalytics: NfLoadLevelInfos is nil or empty")
		return
	}

	nfLoadLevelInfo := eventNotification.NfLoadLevelInfos[0]
	if nfLoadLevelInfo.Confidence == 0 {
		// Confidence is 0, ignore this notification (This notification is not prediction)
		logger.ProcessorLog.Warnln("ReceiveNfLoadLevelAnalytics: Confidence is 0, ignore this notification")
		return
	}

	logger.ProcessorLog.Warnf("ReceiveNfLoadLevelAnalytics: NfLoadLevelInfo: %+v", nfLoadLevelInfo)
	logger.ProcessorLog.Warnf("LoadLevel Peak: %+v", nfLoadLevelInfo.NfLoadLevelpeak)

	p.NfLoadAnalyticsLock.Lock()
	defer p.NfLoadAnalyticsLock.Unlock()

	p.NfLoadAnalytics[nfLoadLevelInfo.NfType] = nfLoadLevelInfo

	// TODO(ctfang): Process the nfLoadLevelInfo
	// If the NfLoadLevelPeak is greater than the threshold, adjust the URR report threshold
}

func (p *Processor) SubscribeNfLoadIfNotExist(ctx context.Context) {
	if p.Config().Configuration.Nwdaf.Enable == false {
		logger.ProcessorLog.Warn("NWDAF is not enabled, no need to subscribe")
		return
	}
	if p.NwdafUri == "" {
		logger.ProcessorLog.Infoln("SubscribeNfLoadIfNotExist(): Try to subscribe to NWDAF")

		p.subscribeNfLoadLevelAnalytics(ctx)
	} else {
		logger.ProcessorLog.Infof("NWDafUri is not empty, no need to subscribe")
	}
}

func (p *Processor) subscribeNfLoadLevelAnalytics(ctx context.Context) {
	if p.NwdafUri == "" {
		p.NwdafLock.Lock()
		defer p.NwdafLock.Unlock()

		// NWDAF NF Discovery
		result, err := p.Consumer().SearchNFInstances(ctx, models.NrfNfManagementNfType_NWDAF, models.NrfNfManagementNfType_SMF, nil)
		if err != nil || result == nil {
			logger.ProcessorLog.Errorf("NWDAF SearchNFInstances failed: %+v, result: %v", err, result)
			return
		}
		if len((*result).NfInstances) == 0 {
			logger.ProcessorLog.Warn("NWDAF SearchNFInstances result is empty")
			return
		}
		nwdafProfile := (*result).NfInstances[0]
		for _, service := range nwdafProfile.NfServices {
			if service.ServiceName == models.ServiceName_NNWDAF_EVENTSSUBSCRIPTION {
				endpoint := service.IpEndPoints[0]
				p.NwdafUri = fmt.Sprintf("%s://%s:%d", service.Scheme, endpoint.Ipv4Address, endpoint.Port)
				break
			}
		}
		if p.NwdafUri == "" {
			logger.ProcessorLog.Warn("NWDAF SearchNFInstances result is empty")
			return
		}
	}

	logger.ProcessorLog.Infof("CreateEventSubscription to %s", p.NwdafUri)

	subscriptionId, err := p.Consumer().CreateEventSubscription(ctx, p.NwdafUri)
	if err != nil {
		logger.ProcessorLog.Errorf("CreateEventSubscription failed: %+v", err)
		return
	}

	logger.ProcessorLog.Infof("CreateEventSubscription success: %s", subscriptionId)
	p.NwdafSubscriptionId = subscriptionId
}

func (p *Processor) deleteSubscriptions(ctx context.Context) {
	p.NwdafLock.Lock()
	defer p.NwdafLock.Unlock()

	if p.NwdafSubscriptionId == "" {
		logger.ProcessorLog.Warn("NwdafSubscriptionId is empty")
		return
	}

	err := p.Consumer().DeleteEventSubscription(ctx, p.NwdafUri, p.NwdafSubscriptionId)
	if err != nil {
		logger.ProcessorLog.Errorf("DeleteEventSubscription failed: %+v", err)
		return
	}
	logger.ProcessorLog.Infof("DeleteEventSubscription success: %s", p.NwdafSubscriptionId)
	p.NwdafSubscriptionId = ""
}

// Check whether the condition is met
// If the condition is met, return true
func (p *Processor) CheckNwdafNfLoadCondition() bool {
	if p.Config().Configuration.Nwdaf.Enable == false {
		return false
	}

	cfg := p.Config().Configuration.Nwdaf
	if cfg.LoadThreshold == 0 && cfg.CpuThreshold == 0 && cfg.MemThreshold == 0 {
		logger.ProcessorLog.Warn("Nwdaf Load Condition is not set")
		return false
	}

	if chfLoad, ok := p.NfLoadAnalytics[models.NrfNfManagementNfType_CHF]; ok {
		if cfg.LoadThreshold != 0 && chfLoad.NfLoadLevelAverage > cfg.LoadThreshold {
			logger.ProcessorLog.Warnf("CHF Load Level is %d", chfLoad.NfLoadLevelAverage)
			return true
		}
		if cfg.CpuThreshold != 0 && chfLoad.NfCpuUsage > cfg.CpuThreshold {
			logger.ProcessorLog.Warnf("CHF CPU Usage is %d", chfLoad.NfCpuUsage)
			return true
		}
		if cfg.MemThreshold != 0 && chfLoad.NfMemoryUsage > cfg.MemThreshold {
			logger.ProcessorLog.Warnf("CHF Memory Usage is %d", chfLoad.NfMemoryUsage)
			return true
		}
	}
	if smfLoad, ok := p.NfLoadAnalytics[models.NrfNfManagementNfType_SMF]; ok {
		if cfg.LoadThreshold != 0 && smfLoad.NfLoadLevelAverage > cfg.LoadThreshold {
			logger.ProcessorLog.Warnf("SMF Load Level is %d", smfLoad.NfLoadLevelAverage)
			return true
		}
		if cfg.CpuThreshold != 0 && smfLoad.NfCpuUsage > cfg.CpuThreshold {
			logger.ProcessorLog.Warnf("SMF CPU Usage is %d", smfLoad.NfCpuUsage)
			return true
		}
		if cfg.MemThreshold != 0 && smfLoad.NfMemoryUsage > cfg.MemThreshold {
			logger.ProcessorLog.Warnf("SMF Memory Usage is %d", smfLoad.NfMemoryUsage)
			return true
		}
	}
	if upfLoad, ok := p.NfLoadAnalytics[models.NrfNfManagementNfType_UPF]; ok {
		if cfg.LoadThreshold != 0 && upfLoad.NfLoadLevelAverage > cfg.LoadThreshold {
			logger.ProcessorLog.Warnf("UPF Load Level is %d", upfLoad.NfLoadLevelAverage)
			return true
		}
		if cfg.CpuThreshold != 0 && upfLoad.NfCpuUsage > cfg.CpuThreshold {
			logger.ProcessorLog.Warnf("UPF CPU Usage is %d", upfLoad.NfCpuUsage)
			return true
		}
		if cfg.MemThreshold != 0 && upfLoad.NfMemoryUsage > cfg.MemThreshold {
			logger.ProcessorLog.Warnf("UPF Memory Usage is %d", upfLoad.NfMemoryUsage)
			return true
		}
	}
	return false
}
