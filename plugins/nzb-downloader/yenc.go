package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// YencDecoder handles yEnc decoding
type YencDecoder struct{}

// Decode decodes yEnc encoded data
func (d *YencDecoder) Decode(data []byte) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))

	var decoded []byte
	inBody := false

	for _, line := range lines {
		// Only trim \r at the end, don't use TrimSpace which might remove valid data
		line = bytes.TrimSuffix(line, []byte("\r"))

		// Check for yEnc header
		if bytes.HasPrefix(line, []byte("=ybegin")) {
			inBody = true
			continue
		}

		// Check for yEnc part line
		if bytes.HasPrefix(line, []byte("=ypart")) {
			continue
		}

		// Check for yEnc end
		if bytes.HasPrefix(line, []byte("=yend")) {
			break
		}

		// Skip empty lines and lines before ybegin
		if !inBody || len(line) == 0 {
			continue
		}

		// Decode the line
		decodedLine := d.decodeLine(line)
		decoded = append(decoded, decodedLine...)
	}

	return decoded, nil
}

// decodeLine decodes a single yEnc line
func (d *YencDecoder) decodeLine(line []byte) []byte {
	var decoded []byte

	i := 0
	for i < len(line) {
		b := line[i]

		// Handle escape character
		if b == '=' {
			i++
			if i >= len(line) {
				break
			}
			b = line[i]
			// Escaped bytes have both offsets applied during encoding
			b = (b - 64 - 42) & 0xFF
		} else {
			b = (b - 42) & 0xFF
		}

		decoded = append(decoded, b)
		i++
	}

	return decoded
}

// ParseYencHeader parses yEnc header information
func ParseYencHeader(data []byte) (filename string, size int64, err error) {
	lines := bytes.Split(data, []byte("\n"))

	for _, line := range lines {
		line = bytes.TrimSpace(line)

		if bytes.HasPrefix(line, []byte("=ybegin")) {
			// Parse parameters
			parts := bytes.Split(line, []byte(" "))
			for _, part := range parts {
				if bytes.HasPrefix(part, []byte("name=")) {
					filename = string(bytes.TrimPrefix(part, []byte("name=")))
				}
				if bytes.HasPrefix(part, []byte("size=")) {
					sizeStr := string(bytes.TrimPrefix(part, []byte("size=")))
					size, err = strconv.ParseInt(sizeStr, 10, 64)
					if err != nil {
						return "", 0, fmt.Errorf("invalid size: %v", err)
					}
				}
			}
			return filename, size, nil
		}
	}

	return "", 0, fmt.Errorf("no yEnc header found")
}

// IsYencEncoded checks if data appears to be yEnc encoded
func IsYencEncoded(data []byte) bool {
	return bytes.Contains(data, []byte("=ybegin"))
}

// DecodeArticle decodes a complete article (with headers)
func DecodeArticle(data []byte) ([]byte, error) {
	// Find where the body starts (after blank line in headers)
	bodyStart := bytes.Index(data, []byte("\r\n\r\n"))
	if bodyStart == -1 {
		bodyStart = bytes.Index(data, []byte("\n\n"))
		if bodyStart == -1 {
			// No headers, assume entire data is body
			bodyStart = 0
		} else {
			bodyStart += 2
		}
	} else {
		bodyStart += 4
	}

	body := data[bodyStart:]

	// Check if it's yEnc encoded
	if !IsYencEncoded(body) {
		return nil, fmt.Errorf("article is not yEnc encoded")
	}

	decoder := &YencDecoder{}
	decoded, err := decoder.Decode(body)
	if err != nil {
		return nil, err
	}

	// Validate we got some data
	if len(decoded) == 0 {
		return nil, fmt.Errorf("yEnc decode produced empty result")
	}

	return decoded, nil
}

// CleanFilename removes yEnc artifacts from filename
func CleanFilename(filename string) string {
	// Remove quotes
	filename = strings.Trim(filename, "\"")

	// Remove yEnc part suffix (e.g., ".001")
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		if _, err := strconv.Atoi(filename[idx+1:]); err == nil {
			filename = filename[:idx]
		}
	}

	return filename
}
