package sbi

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/free5gc/nwdaf/pkg/models"
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
	}
}

func (s *Server) SmfOamNfResourceGet(c *gin.Context) {
	nfResource, err := GetNfResouces(context.Background())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, *nfResource)
}

func GetNfResouces(ctx context.Context) (*models.NfResourceUsage, error) {
	var memoryCurrent []byte
	var memoryMax []byte
	var errMem error

	// Read memory usage from cgroup
	memoryCurrent, errMem = os.ReadFile("/sys/fs/cgroup/memory.current")
	if errMem != nil {
		memoryCurrent = []byte("1")
	}
	memoryMax, errMem = os.ReadFile("/sys/fs/cgroup/memory.max")
	if errMem != nil {
		memoryMax = []byte("1")
	}

	// Convert memory values to integers
	currentMemory, err := strconv.ParseUint(strings.TrimSpace(string(memoryCurrent)), 10, 64)
	if err != nil {
		return nil, err
	}
	maxMemory, err := strconv.ParseUint(strings.TrimSpace(string(memoryMax)), 10, 64)
	if err != nil {
		return nil, err
	}

	// Calculate memory usage percentage
	memoryUsage := float64(currentMemory) / float64(maxMemory) * 100

	pid := os.Getpid()
	proc, procErr := process.NewProcess(int32(pid))
	if procErr != nil {
		return nil, procErr
	}
	cpuUsage, cpuErr := proc.Percent(time.Second)
	if cpuErr != nil {
		return nil, cpuErr
	}

	numGoruntine := int32(runtime.NumGoroutine())

	return &models.NfResourceUsage{
		Time:         time.Now(),
		TotalMemory:  maxMemory,
		FreeMemory:   maxMemory - currentMemory,
		MemoryUsage:  memoryUsage,
		CpuUsage:     cpuUsage,
		NumGoroutine: numGoruntine,
	}, nil
}
