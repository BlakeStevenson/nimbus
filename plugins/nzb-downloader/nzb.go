package main

import (
	"encoding/xml"
	"io"
	"strings"
)

// NZB represents the root element of an NZB file
type NZB struct {
	XMLName xml.Name  `xml:"nzb"`
	Files   []NZBFile `xml:"file"`
}

// NZBFile represents a file in the NZB
type NZBFile struct {
	Poster   string       `xml:"poster,attr"`
	Date     int64        `xml:"date,attr"`
	Subject  string       `xml:"subject,attr"`
	FileName string       `xml:"http://www.newzbin.com/DTD/2003/nzb name,attr"`
	Groups   []string     `xml:"groups>group"`
	Segments []NZBSegment `xml:"segments>segment"`
}

// NZBSegment represents a segment of a file
type NZBSegment struct {
	Bytes     int64  `xml:"bytes,attr"`
	Number    int    `xml:"number,attr"`
	MessageID string `xml:",chardata"`
}

// Filename extracts the filename from a subject line
func (f *NZBFile) Filename() string {
	// First, check if there's a name attribute (modern NZB format)
	if f.FileName != "" {
		return strings.TrimSpace(f.FileName)
	}

	// Try to extract filename from subject
	// Common Usenet subject formats:
	// - "filename.ext" yEnc (1/10)
	// - filename.ext (1/10) "filename.ext" yEnc
	// - [12345] filename.ext (1/10)
	// - filename.ext [1/10] - "filename.ext" yEnc
	// - [PREFIX]-[GROUP]-[filename.ext]-[N/M]
	subject := strings.TrimSpace(f.Subject)
	if subject == "" {
		return ""
	}

	// Strategy 0: Look for bracket-enclosed filename (common in private indexers)
	// Format: [PREFIX]-[GROUP]-[filename.ext]-[N/M]
	// Search all bracketed sections and find the one that looks like a filename
	if strings.Contains(subject, "[") && strings.Contains(subject, "]") {
		start := -1
		for i := 0; i < len(subject); i++ {
			if subject[i] == '[' {
				start = i + 1
			} else if subject[i] == ']' && start != -1 {
				candidate := subject[start:i]
				candidate = strings.TrimSpace(candidate)
				// Check if this looks like a filename (has an extension and is reasonably long)
				if len(candidate) > 5 && strings.Contains(candidate, ".") &&
					!strings.Contains(candidate, "/") && !strings.Contains(candidate, "\\") &&
					(strings.HasSuffix(candidate, ".rar") ||
						strings.HasSuffix(candidate, ".nfo") ||
						strings.HasSuffix(candidate, ".sfv") ||
						strings.HasSuffix(candidate, ".mkv") ||
						strings.Contains(candidate, ".r")) { // .r00, .r01, etc.
					return candidate
				}
				start = -1
			}
		}
	}

	// Strategy 1: Look for quoted filename (most reliable)
	// Find the last quoted string as it's usually the actual filename
	lastQuoteStart := -1
	lastQuoteEnd := -1
	inQuote := false
	for i := 0; i < len(subject); i++ {
		if subject[i] == '"' {
			if !inQuote {
				lastQuoteStart = i
				inQuote = true
			} else {
				lastQuoteEnd = i
				inQuote = false
			}
		}
	}

	if lastQuoteStart != -1 && lastQuoteEnd > lastQuoteStart+1 {
		filename := subject[lastQuoteStart+1 : lastQuoteEnd]
		filename = strings.TrimSpace(filename)
		// Validate it looks like a filename (has extension and not just numbers)
		if strings.Contains(filename, ".") && !strings.HasPrefix(filename, ".") {
			// Make sure it's not just a part indicator like "1/10"
			if !strings.Contains(filename, "/") || strings.LastIndex(filename, ".") > strings.Index(filename, "/") {
				return filename
			}
		}
	}

	// Strategy 2: Look for pattern with yEnc marker
	if idx := strings.Index(subject, " yEnc"); idx != -1 {
		beforeYenc := strings.TrimSpace(subject[:idx])

		// Remove part indicators like (1/10) [1/10] from the end
		beforeYenc = removeSuffixPattern(beforeYenc, `\([0-9]+/[0-9]+\)`)
		beforeYenc = removeSuffixPattern(beforeYenc, `\[[0-9]+/[0-9]+\]`)
		beforeYenc = strings.TrimSpace(beforeYenc)

		// Try to find filename by looking for the last token with a file extension
		parts := strings.Fields(beforeYenc)
		for i := len(parts) - 1; i >= 0; i-- {
			part := parts[i]
			// Skip bracketed content
			if strings.HasPrefix(part, "[") || strings.HasPrefix(part, "(") {
				continue
			}
			// Remove trailing punctuation
			part = strings.TrimRight(part, ".,;:!?")
			// Check if it looks like a filename
			if strings.Contains(part, ".") && !strings.HasPrefix(part, ".") {
				// Validate it has a reasonable extension
				if lastDot := strings.LastIndex(part, "."); lastDot > 0 && lastDot < len(part)-1 {
					ext := part[lastDot+1:]
					// Extension should be 2-4 chars typically
					if len(ext) >= 2 && len(ext) <= 4 {
						return part
					}
				}
			}
		}
	}

	// Strategy 3: Find any token that looks like a filename
	// Look for something with a file extension that's not in brackets
	parts := strings.Fields(subject)
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		// Skip bracketed/parenthesized parts and part indicators
		if strings.HasPrefix(part, "[") || strings.HasPrefix(part, "(") {
			continue
		}
		if strings.Contains(part, "/") && strings.Count(part, "/") == 1 {
			// Likely a part indicator like 1/10
			continue
		}
		// Remove trailing punctuation
		part = strings.TrimRight(part, ".,;:!?()")
		// Check if it looks like a filename with extension
		if lastDot := strings.LastIndex(part, "."); lastDot > 0 && lastDot < len(part)-1 {
			ext := part[lastDot+1:]
			// Common extensions or reasonable length
			commonExts := []string{"mkv", "mp4", "avi", "rar", "zip", "nzb", "par2", "vol", "r00", "r01"}
			isCommon := false
			for _, ce := range commonExts {
				if strings.HasPrefix(strings.ToLower(ext), ce) {
					isCommon = true
					break
				}
			}
			if isCommon || (len(ext) >= 2 && len(ext) <= 4) {
				return part
			}
		}
	}

	return ""
}

// removeSuffixPattern removes pattern from end of string (simple version without regex)
func removeSuffixPattern(s string, pattern string) string {
	// Simple heuristic: remove common part indicators from the end
	s = strings.TrimSpace(s)
	for {
		orig := s
		// Remove (x/y) pattern
		if idx := strings.LastIndex(s, "("); idx != -1 {
			if eidx := strings.LastIndex(s, ")"); eidx > idx && eidx == len(s)-1 {
				inside := s[idx+1 : eidx]
				// Check if it looks like a part indicator
				if strings.Contains(inside, "/") && len(strings.Split(inside, "/")) == 2 {
					s = strings.TrimSpace(s[:idx])
					continue
				}
			}
		}
		// Remove [x/y] pattern
		if idx := strings.LastIndex(s, "["); idx != -1 {
			if eidx := strings.LastIndex(s, "]"); eidx > idx && eidx == len(s)-1 {
				inside := s[idx+1 : eidx]
				// Check if it looks like a part indicator
				if strings.Contains(inside, "/") && len(strings.Split(inside, "/")) == 2 {
					s = strings.TrimSpace(s[:idx])
					continue
				}
			}
		}
		if orig == s {
			break
		}
	}
	return s
}

// ParseNZB parses an NZB file from a reader
func ParseNZB(r io.Reader) (*NZB, error) {
	var nzb NZB
	decoder := xml.NewDecoder(r)

	if err := decoder.Decode(&nzb); err != nil {
		return nil, err
	}

	return &nzb, nil
}

// TotalBytes calculates the total size of all files
func (n *NZB) TotalBytes() int64 {
	var total int64
	for _, file := range n.Files {
		for _, seg := range file.Segments {
			total += seg.Bytes
		}
	}
	return total
}
