package boxapi

import (
	"context"

	"github.com/sagernet/sing-box/experimental/v2rayapi"
	"github.com/sagernet/sing-box/option"
)

// SbStatsService preserves the Neko query API while using sing-box accounting.
type SbStatsService struct {
	*v2rayapi.StatsService
}

func NewSbStatsService(options option.V2RayStatsServiceOptions) *SbStatsService {
	stats := v2rayapi.NewStatsService(options)
	if stats == nil {
		return nil
	}
	return &SbStatsService{StatsService: stats}
}

func (s *SbStatsService) GetStats(ctx context.Context, name string, reset bool) (int64, error) {
	response, err := s.StatsService.GetStats(ctx, &v2rayapi.GetStatsRequest{
		Name:   name,
		Reset_: reset,
	})
	if err != nil {
		return 0, err
	}
	return response.GetStat().GetValue(), nil
}
