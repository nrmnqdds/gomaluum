package server

import (
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

// @Title AdsHandler
// @Description Get i-Ma'luum ads
// @Tags scraper
// @Produce json
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/ads [get]
func (s *Server) AdsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	logger := s.log.GetLogger()
	ads := []dtos.Ads{}

	c := colly.NewCollector(
		colly.Headers(
			map[string]string{
				"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
				"Accept-Language": "en-US,en;q=0.9",
				"Cookie":          "lang=en; theme=light;",
			},
		),
		colly.UserAgent(constants.UserAgent),
	)
	c.WithTransport(s.httpClient.Transport)

	c.OnHTML("div[style*='width:100%; clear:both;height:100px']", func(e *colly.HTMLElement) {
		ads = append(ads, dtos.Ads{
			Title:    strings.TrimSpace(e.ChildText("a")),
			ImageURL: strings.TrimSpace(e.ChildAttr("img", "src")),
			Link:     strings.TrimSpace(e.ChildAttr("a", "href")),
			ID:       cuid.New(),
		})
	})

	if err := c.Visit("https://souq.iium.edu.my/embeded"); err != nil {
		logger.Sugar().Errorf("Failed to visit ads page: %v", err)
		errors.Render(w, r, errors.ErrFailedToGoToURL)
		return
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched ads",
		Data:    &ads,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
