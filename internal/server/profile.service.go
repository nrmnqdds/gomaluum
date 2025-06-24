package server

import (
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

// Object pool for profile data processing
var profileDataPool = sync.Pool{
	New: func() any {
		return &profileData{}
	},
}

// Struct to hold temporary profile data for efficient processing
type profileData struct {
	name          string
	matricNo      string
	level         string
	kuliyyah      string
	ic            string
	gender        string
	birthday      string
	religion      string
	maritalStatus string
	address       string
}

// Pre-defined selectors for better performance
var profileSelectors = struct {
	name          string
	matricNo      string
	level         string
	kuliyyah      string
	ic            string
	gender        string
	birthday      string
	religion      string
	maritalStatus string
	address       string
}{
	name:          ".row .col-md-12 .box.box-default .panel-body.row .col-md-4[style='text-align:center; padding:10px; floaf:left;'] h4[style='margin-top:1%;']",
	matricNo:      ".row .col-md-12 .box.box-default .panel-body.row .col-md-4[style='margin-top:3%;'] h4",
	level:         ".row .col-md-12 .box.box-default .panel-body.row .col-md-4[style='text-align:center; padding:10px; floaf:left;'] h4:nth-of-type(2)",
	kuliyyah:      ".row .col-md-12 .box.box-default .panel-body.row .col-md-4[style='margin-top:3%;'] p",
	ic:            ".row:nth-of-type(2) .col-md-12 .nav-tabs-custom .tab-content .tab-pane.active .row .col-md-3 p:nth-of-type(2)",
	gender:        ".row:nth-of-type(2) .col-md-12 .nav-tabs-custom .tab-content .tab-pane.active .row .col-md-3 p:nth-of-type(3)",
	birthday:      ".row:nth-of-type(2) .col-md-12 .nav-tabs-custom .tab-content .tab-pane.active .row .col-md-3 p:nth-of-type(4)",
	religion:      ".row:nth-of-type(2) .col-md-12 .nav-tabs-custom .tab-content .tab-pane.active .row .col-md-3 p:nth-of-type(5)",
	maritalStatus: ".row:nth-of-type(2) .col-md-12 .nav-tabs-custom .tab-content .tab-pane.active .row .col-md-9 p:nth-of-type(2)",
	address:       ".row:nth-of-type(2) .col-md-12 .nav-tabs-custom .tab-content .tab-pane.active .row .col-md-9 p:nth-of-type(3)",
}

// Efficiently extract field value from colon-separated text
func extractFieldValue(text string) string {
	if idx := strings.Index(text, ":"); idx != -1 && len(text) > idx+1 {
		return strings.TrimSpace(text[idx+1:])
	}
	return strings.TrimSpace(text)
}

// Extract matric number from pipe-separated text
func extractMatricNo(text string) string {
	if idx := strings.Index(text, "|"); idx != -1 {
		return strings.TrimSpace(text[:idx])
	}
	return strings.TrimSpace(text)
}

// Format address efficiently by cleaning and joining lines
func formatAddress(address string) string {
	if address == "" {
		return ""
	}

	lines := strings.Split(address, "\n")
	formattedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		cleaned := strings.TrimSpace(strings.ReplaceAll(line, "\t", " "))
		// Replace multiple spaces with single space
		for strings.Contains(cleaned, "  ") {
			cleaned = strings.ReplaceAll(cleaned, "  ", " ")
		}
		if cleaned != "" {
			formattedLines = append(formattedLines, cleaned)
		}
	}

	return strings.Join(formattedLines, ", ")
}

// Build image URL efficiently
func buildImageURL(matricNo string) string {
	const baseURL = "https://smartcard.iium.edu.my/packages/card/printing/camera/uploads/original/"
	const extension = ".jpeg"

	// Pre-calculate capacity to avoid reallocation
	capacity := len(baseURL) + len(matricNo) + len(extension)
	var builder strings.Builder
	builder.Grow(capacity)

	builder.WriteString(baseURL)
	builder.WriteString(matricNo)
	builder.WriteString(extension)

	return builder.String()
}

// Extract and process profile data efficiently
func extractProfileData(e *colly.HTMLElement) *profileData {
	data := profileDataPool.Get().(*profileData)
	// Reset the struct
	*data = profileData{}

	// Extract all data in a single DOM traversal
	data.name = strings.TrimSpace(e.ChildText(profileSelectors.name))
	data.matricNo = extractMatricNo(strings.TrimSpace(e.ChildText(profileSelectors.matricNo)))
	data.level = strings.TrimSpace(e.ChildText(profileSelectors.level))
	data.kuliyyah = strings.TrimSpace(e.ChildText(profileSelectors.kuliyyah))
	data.ic = extractFieldValue(strings.TrimSpace(e.ChildText(profileSelectors.ic)))
	data.gender = extractFieldValue(strings.TrimSpace(e.ChildText(profileSelectors.gender)))
	data.birthday = extractFieldValue(strings.TrimSpace(e.ChildText(profileSelectors.birthday)))
	data.religion = extractFieldValue(strings.TrimSpace(e.ChildText(profileSelectors.religion)))
	data.maritalStatus = extractFieldValue(strings.TrimSpace(e.ChildText(profileSelectors.maritalStatus)))

	// Handle address separately as it needs special formatting
	addressText := strings.TrimSpace(e.ChildText(profileSelectors.address))
	addressValue := extractFieldValue(addressText)
	data.address = formatAddress(addressValue)

	return data
}

func (s *Server) Profile(cookie string) (*dtos.Profile, error) {
	logger := s.log.GetLogger()

	// Pre-build cookie string
	cookieStr := "MOD_AUTH_CAS=" + cookie

	c := colly.NewCollector()
	c.WithTransport(s.httpClient.Transport)

	var profileResult *dtos.Profile

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", cookieStr)
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML("body", func(e *colly.HTMLElement) {
		// Extract all profile data efficiently
		data := extractProfileData(e)
		defer profileDataPool.Put(data)

		// Build the profile response
		profileResult = &dtos.Profile{
			Name:          data.name,
			MatricNo:      data.matricNo,
			Level:         data.level,
			Kuliyyah:      data.kuliyyah,
			IC:            data.ic,
			Gender:        data.gender,
			Birthday:      data.birthday,
			Religion:      data.religion,
			MaritalStatus: data.maritalStatus,
			Address:       data.address,
			ImageURL:      buildImageURL(data.matricNo),
		}
	})

	if err := c.Visit(constants.ImaluumProfilePage); err != nil {
		logger.Sugar().Errorf("Failed to go to URL: %v", err)
		return nil, errors.ErrFailedToGoToURL
	}

	if profileResult == nil {
		logger.Sugar().Error("Failed to extract profile data")
		return nil, errors.ErrFailedToGoToURL
	}

	return profileResult, nil
}
