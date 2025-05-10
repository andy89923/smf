package processor

import (
	"context"
	"sync"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/sbi/consumer"
	"github.com/free5gc/smf/pkg/app"
)

const (
	CONTEXT_NOT_FOUND = "CONTEXT_NOT_FOUND"
)

type ProcessorSmf interface {
	app.App

	Consumer() *consumer.Consumer
}

type Processor struct {
	ProcessorSmf

	UrrLock              sync.Mutex
	UrrReportCount       int
	ChargingUrrThreshold uint64

	// Following fields are used for NWDAF subscription
	NwdafLock           sync.Mutex
	NwdafUri            string
	NwdafSubscriptionId string

	// NfLoadAnalytics
	NfLoadAnalyticsLock sync.Mutex
	NfLoadAnalytics     map[models.NrfNfManagementNfType]models.NfLoadLevelInformation
}

func NewProcessor(smf ProcessorSmf) (*Processor, error) {
	p := &Processor{
		ProcessorSmf:         smf,
		UrrReportCount:       0,
		ChargingUrrThreshold: smf.Config().Configuration.UrrThreshold,
		NwdafUri:             "",
		NwdafSubscriptionId:  "",
		NfLoadAnalytics:      make(map[models.NrfNfManagementNfType]models.NfLoadLevelInformation),
	}
	return p, nil
}

func (p *Processor) Stop(ctx context.Context) {
	if p.NwdafSubscriptionId != "" {
		p.deleteSubscriptions(ctx)
		p.NwdafSubscriptionId = ""
	}
}
