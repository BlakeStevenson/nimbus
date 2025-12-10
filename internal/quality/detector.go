package quality

import (
	"regexp"
	"strings"
)

// Detector handles quality detection from release names
type Detector struct {
	resolutionRegex *regexp.Regexp
	sourceRegex     *regexp.Regexp
	codecVideoRegex *regexp.Regexp
	codecAudioRegex *regexp.Regexp
	modifierRegex   *regexp.Regexp
}

// NewDetector creates a new quality detector
func NewDetector() *Detector {
	return &Detector{
		resolutionRegex: regexp.MustCompile(`(?i)(480|576|720|1080|2160)[pi]?`),
		sourceRegex:     regexp.MustCompile(`(?i)(blu[-\s]?ray|bluray|brrip|bdrip|bd|web[-\s]?dl|webdl|web[-\s]?rip|webrip|hdtv|sdtv|dvd[-\s]?rip|dvdrip|dvd|pdtv|hdcam|cam|ts|telesync|r5|screener)`),
		codecVideoRegex: regexp.MustCompile(`(?i)(x264|x265|h\.?264|h\.?265|hevc|avc|mpeg[-\s]?2|mpeg2|xvid|divx|av1|vp9)`),
		codecAudioRegex: regexp.MustCompile(`(?i)(atmos|truehd|dts[-\s]?hd|dtshd|dts[-\s]?x|dtsx|dts|dd5\.1|dd\+?5\.1|ac3|aac|mp3|flac|opus|pcm)`),
		modifierRegex:   regexp.MustCompile(`(?i)(remux|proper|repack|remastered|extended|unrated|directors?\.cut|theatrical|imax)`),
	}
}

// DetectQuality detects quality information from a release name
func (d *Detector) DetectQuality(releaseName string) *DetectedQualityInfo {
	info := &DetectedQualityInfo{
		QualityName: "Unknown",
	}

	// Normalize the release name
	normalized := strings.ToLower(releaseName)

	// Detect resolution
	if matches := d.resolutionRegex.FindStringSubmatch(releaseName); len(matches) > 1 {
		resolutionStr := matches[1]
		var resolution int
		switch resolutionStr {
		case "480":
			resolution = 480
		case "576":
			resolution = 576
		case "720":
			resolution = 720
		case "1080":
			resolution = 1080
		case "2160":
			resolution = 2160
		}
		info.Resolution = &resolution
	}

	// Detect source
	if matches := d.sourceRegex.FindStringSubmatch(releaseName); len(matches) > 1 {
		sourceStr := normalizeSource(matches[1])
		info.Source = &sourceStr
	}

	// Detect video codec
	if matches := d.codecVideoRegex.FindStringSubmatch(releaseName); len(matches) > 1 {
		codecStr := normalizeCodec(matches[1])
		info.CodecVideo = &codecStr
	}

	// Detect audio codec
	if matches := d.codecAudioRegex.FindStringSubmatch(releaseName); len(matches) > 1 {
		codecStr := normalizeAudioCodec(matches[1])
		info.CodecAudio = &codecStr
	}

	// Detect modifiers
	if strings.Contains(normalized, "remux") {
		info.IsRemux = true
	}
	if strings.Contains(normalized, "proper") {
		info.IsProper = true
	}
	if strings.Contains(normalized, "repack") {
		info.IsRepack = true
	}
	if strings.Contains(normalized, "remastered") {
		info.IsRemastered = true
	}

	// Build quality name based on detection
	info.QualityName = d.buildQualityName(info)

	return info
}

// buildQualityName constructs a quality name from detected info
func (d *Detector) buildQualityName(info *DetectedQualityInfo) string {
	if info.Source == nil || info.Resolution == nil {
		return "Unknown"
	}

	var parts []string

	// Add source
	parts = append(parts, *info.Source)

	// Add resolution
	resStr := ""
	switch *info.Resolution {
	case 480:
		resStr = "480p"
	case 576:
		resStr = "576p"
	case 720:
		resStr = "720p"
	case 1080:
		resStr = "1080p"
	case 2160:
		resStr = "2160p"
	}
	if resStr != "" {
		parts = append(parts, resStr)
	}

	qualityName := strings.Join(parts, "-")

	// Handle remux specially
	if info.IsRemux {
		qualityName = "Remux-" + resStr
	}

	return qualityName
}

// MatchQualityDefinition matches detected quality to a quality definition
func (d *Detector) MatchQualityDefinition(info *DetectedQualityInfo, definitions []QualityDefinition) *QualityDefinition {
	// Try exact name match first
	for i := range definitions {
		if definitions[i].Name == info.QualityName {
			return &definitions[i]
		}
	}

	// Try matching by resolution and source
	if info.Resolution != nil && info.Source != nil {
		for i := range definitions {
			if definitions[i].Resolution != nil && *definitions[i].Resolution == *info.Resolution {
				if definitions[i].Source != nil && strings.EqualFold(*definitions[i].Source, *info.Source) {
					return &definitions[i]
				}
			}
		}
	}

	// Try matching by resolution only
	if info.Resolution != nil {
		var bestMatch *QualityDefinition
		for i := range definitions {
			if definitions[i].Resolution != nil && *definitions[i].Resolution == *info.Resolution {
				if bestMatch == nil || definitions[i].Weight > bestMatch.Weight {
					bestMatch = &definitions[i]
				}
			}
		}
		if bestMatch != nil {
			return bestMatch
		}
	}

	// Return Unknown quality if exists
	for i := range definitions {
		if definitions[i].Name == "Unknown" {
			return &definitions[i]
		}
	}

	return nil
}

// normalizeSource normalizes source strings
func normalizeSource(source string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(source, "-", ""), " ", ""))

	switch {
	case strings.Contains(normalized, "BLURAY"), strings.Contains(normalized, "BRRIP"),
		strings.Contains(normalized, "BDRIP"), normalized == "BD":
		return "BLURAY"
	case strings.Contains(normalized, "WEBDL"):
		return "WEBDL"
	case strings.Contains(normalized, "WEBRIP"):
		return "WEBRIP"
	case strings.Contains(normalized, "HDTV"):
		return "HDTV"
	case strings.Contains(normalized, "SDTV"), strings.Contains(normalized, "PDTV"):
		return "TV"
	case strings.Contains(normalized, "DVDRIP"), strings.Contains(normalized, "DVD"):
		return "DVD"
	case strings.Contains(normalized, "CAM"), strings.Contains(normalized, "HDCAM"):
		return "CAM"
	case strings.Contains(normalized, "TS"), strings.Contains(normalized, "TELESYNC"):
		return "TELESYNC"
	case strings.Contains(normalized, "SCREENER"), normalized == "R5":
		return "SCREENER"
	default:
		return normalized
	}
}

// normalizeCodec normalizes video codec strings
func normalizeCodec(codec string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(codec, ".", ""), " ", ""))

	switch {
	case normalized == "X264", normalized == "H264", normalized == "AVC":
		return "H.264"
	case normalized == "X265", normalized == "H265", normalized == "HEVC":
		return "H.265"
	case normalized == "AV1":
		return "AV1"
	case normalized == "VP9":
		return "VP9"
	case strings.Contains(normalized, "XVID"):
		return "XviD"
	case strings.Contains(normalized, "DIVX"):
		return "DivX"
	case strings.Contains(normalized, "MPEG2"):
		return "MPEG-2"
	default:
		return normalized
	}
}

// normalizeAudioCodec normalizes audio codec strings
func normalizeAudioCodec(codec string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(codec, ".", ""), " ", ""))

	switch {
	case strings.Contains(normalized, "ATMOS"):
		return "Atmos"
	case strings.Contains(normalized, "TRUEHD"):
		return "TrueHD"
	case strings.Contains(normalized, "DTSHD"):
		return "DTS-HD"
	case strings.Contains(normalized, "DTSX"):
		return "DTS-X"
	case normalized == "DTS":
		return "DTS"
	case strings.Contains(normalized, "DD51"), strings.Contains(normalized, "DD+51"), normalized == "AC3":
		return "DD5.1"
	case normalized == "AAC":
		return "AAC"
	case normalized == "MP3":
		return "MP3"
	case normalized == "FLAC":
		return "FLAC"
	case normalized == "OPUS":
		return "Opus"
	case normalized == "PCM":
		return "PCM"
	default:
		return normalized
	}
}
