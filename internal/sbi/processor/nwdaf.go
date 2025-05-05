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

	// TODO: Process the nfLoadLevelInfo
	// If the NfLoadLevelPeak is greater than the threshold, adjust the token period
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
