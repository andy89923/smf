package processor

import (
	"sync"

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
}

func NewProcessor(smf ProcessorSmf) (*Processor, error) {
	p := &Processor{
		ProcessorSmf:         smf,
		UrrReportCount:       0,
		ChargingUrrThreshold: smf.Config().Configuration.UrrThreshold,
	}
	return p, nil
}
