package library

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// =============================================================================
// ParsedMedia - Structured representation of parsed filename information
// =============================================================================
// This struct holds all the metadata extracted from a filename or directory path.
// The parser attempts to identify the media type and extract relevant fields.
//
// Supported media types:
//   - movie: Standalone video files with optional year
//   - tv_episode: TV show episodes with season/episode numbers
//   - music_track: Audio files within Artist/Album directory structure
//   - book: eBook files with optional author information
//
// All fields are optional and will be empty/zero if not detected.
// =============================================================================

type ParsedMedia struct {
	Kind    string // "movie", "tv_episode", "music_track", "book", etc.
	Title   string // Main title (movie name, show name, track name, book title)
	Year    int    // Release year (primarily for movies)
	Season  int    // TV season number
	Episode int    // TV episode number
	Artist  string // Music artist name
	Album   string // Music album name
	Track   int    // Music track number
	Author  string // Book author
}

// =============================================================================
// Common Regex Patterns for Filename Parsing
// =============================================================================

var (
	// Movie patterns - match year in various formats
	// Examples: "Movie.Name.2021.mkv", "Movie Name (2021).mp4", "Movie.Name[2021].avi"
	movieYearPattern = regexp.MustCompile(`[\[\(]?(\d{4})[\]\)]?`)

	// TV show patterns - match season and episode numbers
	// Examples: "S01E02", "s01e02", "1x02", "Season 1 Episode 2"
	tvSeasonEpisodePattern1 = regexp.MustCompile(`[Ss](\d{1,2})[Ee](\d{1,2})`)                    // S01E02
	tvSeasonEpisodePattern2 = regexp.MustCompile(`(\d{1,2})[xX](\d{1,2})`)                        // 1x02
	tvSeasonEpisodePattern3 = regexp.MustCompile(`[Ss]eason\s*(\d{1,2}).*[Ee]pisode\s*(\d{1,2})`) // Season 1 Episode 2

	// Music track number pattern
	// Examples: "01 Track Name.mp3", "1. Track Name.flac"
	trackNumberPattern = regexp.MustCompile(`^(\d{1,3})[\s\.\-_]+`)

	// Book author pattern (after dash or hyphen)
	// Examples: "Book Title - Author Name.epub"
	bookAuthorPattern = regexp.MustCompile(`^(.+?)\s*[-–]\s*(.+)$`)

	// Common video extensions
	videoExtensions = map[string]bool{
		".mkv": true, ".mp4": true, ".avi": true, ".mov": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
		".mpg": true, ".mpeg": true, ".m2ts": true, ".ts": true,
	}

	// Common audio extensions
	audioExtensions = map[string]bool{
		".mp3": true, ".flac": true, ".m4a": true, ".aac": true,
		".ogg": true, ".opus": true, ".wma": true, ".wav": true,
		".ape": true, ".alac": true,
	}

	// Common book extensions
	bookExtensions = map[string]bool{
		".epub": true, ".mobi": true, ".azw": true, ".azw3": true,
		".pdf": true, ".djvu": true, ".fb2": true, ".cbz": true,
		".cbr": true,
	}

	// Words to strip from titles for cleaner matching
	stripWords = []string{
		"1080p", "720p", "480p", "2160p", "4k", "bluray", "brrip",
		"webrip", "web-dl", "hdtv", "dvdrip", "xvid", "x264", "x265",
		"hevc", "h264", "h265", "aac", "ac3", "dts", "proper", "repack",
		"extended", "unrated", "directors.cut", "limited", "internal",
	}
)

// =============================================================================
// ParseFilename - Main entry point for filename parsing
// =============================================================================
// Takes an absolute file path and returns a ParsedMedia struct with extracted
// metadata. The function:
//   1. Determines media type from file extension and directory structure
//   2. Applies type-specific parsing rules
//   3. Cleans and normalizes extracted data
//
// Returns nil if the file type is not recognized or cannot be parsed.
// =============================================================================

func ParseFilename(path string) *ParsedMedia {
	ext := strings.ToLower(filepath.Ext(path))
	basename := filepath.Base(path)
	nameWithoutExt := strings.TrimSuffix(basename, ext)
	dir := filepath.Dir(path)

	// Determine media type based on extension
	if videoExtensions[ext] {
		// Check if it's a TV show or movie
		if parsed := parseTVEpisode(nameWithoutExt, dir); parsed != nil {
			return parsed
		}
		return parseMovie(nameWithoutExt)
	}

	if audioExtensions[ext] {
		return parseMusicTrack(nameWithoutExt, dir)
	}

	if bookExtensions[ext] {
		return parseBook(nameWithoutExt)
	}

	return nil
}

// =============================================================================
// parseMovie - Extract movie title and year
// =============================================================================
// Parsing strategy:
//   1. Look for 4-digit year in parentheses, brackets, or standalone
//   2. Everything before the year becomes the title
//   3. Clean up common quality/source tags
//   4. Normalize dots, underscores to spaces
//
// Examples:
//   "The.Dark.Knight.2008.1080p.BluRay.mkv" -> "The Dark Knight" (2008)
//   "Inception (2010).mp4" -> "Inception" (2010)
// =============================================================================

func parseMovie(filename string) *ParsedMedia {
	parsed := &ParsedMedia{
		Kind: "movie",
	}

	// Remove quality and release info
	cleaned := cleanFilename(filename)

	// Try to find year
	if matches := movieYearPattern.FindStringSubmatch(cleaned); len(matches) > 1 {
		year, _ := strconv.Atoi(matches[1])
		parsed.Year = year

		// Title is everything before the year
		yearIndex := strings.Index(cleaned, matches[0])
		if yearIndex > 0 {
			parsed.Title = normalizeTitle(cleaned[:yearIndex])
		}
	}

	// If no year found, use entire cleaned filename as title
	if parsed.Title == "" {
		parsed.Title = normalizeTitle(cleaned)
	}

	return parsed
}

// =============================================================================
// parseTVEpisode - Extract TV show name, season, and episode numbers
// =============================================================================
// Parsing strategy:
//   1. Try multiple regex patterns for S01E02, 1x02, etc.
//   2. Show name is everything before the season/episode marker
//   3. Check parent directories for show name if needed
//
// Examples:
//   "Breaking.Bad.S01E02.mkv" -> "Breaking Bad" S01E02
//   "The Office - 2x05 - Episode Title.mp4" -> "The Office" S02E05
//   "Show/Season 1/Episode 02.mkv" -> "Show" S01E02
// =============================================================================

func parseTVEpisode(filename, dir string) *ParsedMedia {
	cleaned := cleanFilename(filename)

	// Try different season/episode patterns
	patterns := []*regexp.Regexp{
		tvSeasonEpisodePattern1,
		tvSeasonEpisodePattern2,
		tvSeasonEpisodePattern3,
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(cleaned); len(matches) >= 3 {
			season, _ := strconv.Atoi(matches[1])
			episode, _ := strconv.Atoi(matches[2])

			parsed := &ParsedMedia{
				Kind:    "tv_episode",
				Season:  season,
				Episode: episode,
			}

			// Show name is everything before the season/episode marker
			episodeIndex := pattern.FindStringIndex(cleaned)
			if episodeIndex != nil && episodeIndex[0] > 0 {
				parsed.Title = normalizeTitle(cleaned[:episodeIndex[0]])
			}

			// If no title in filename, try to get it from parent directory
			if parsed.Title == "" {
				parsed.Title = getShowNameFromPath(dir)
			}

			return parsed
		}
	}

	// Check if we're in a "Season X" directory structure
	if strings.Contains(strings.ToLower(dir), "season") {
		if season := extractSeasonFromPath(dir); season > 0 {
			// Try to extract episode number from filename
			if matches := regexp.MustCompile(`[Ee]pisode\s*(\d{1,2})`).FindStringSubmatch(filename); len(matches) > 1 {
				episode, _ := strconv.Atoi(matches[1])
				return &ParsedMedia{
					Kind:    "tv_episode",
					Title:   getShowNameFromPath(dir),
					Season:  season,
					Episode: episode,
				}
			}
		}
	}

	return nil
}

// =============================================================================
// parseMusicTrack - Extract artist, album, track number, and track name
// =============================================================================
// Parsing strategy:
//   1. Use directory structure: Artist/Album/Track.mp3
//   2. Extract track number from beginning of filename
//   3. Remaining filename becomes track title
//
// Examples:
//   "Artist/Album/01 Track Name.mp3" -> Artist: "Artist", Album: "Album", Track: 1, Title: "Track Name"
//   "Various Artists/Best Of/05. Song.flac" -> Track 5
// =============================================================================

func parseMusicTrack(filename, dir string) *ParsedMedia {
	parsed := &ParsedMedia{
		Kind: "music_track",
	}

	// Extract track number from filename
	if matches := trackNumberPattern.FindStringSubmatch(filename); len(matches) > 1 {
		track, _ := strconv.Atoi(matches[1])
		parsed.Track = track
		parsed.Title = normalizeTitle(trackNumberPattern.ReplaceAllString(filename, ""))
	} else {
		parsed.Title = normalizeTitle(filename)
	}

	// Try to get artist and album from directory structure
	// Expected: /path/to/Artist/Album/Track.mp3
	parts := strings.Split(dir, string(filepath.Separator))
	if len(parts) >= 2 {
		parsed.Album = parts[len(parts)-1]
		parsed.Artist = parts[len(parts)-2]
	} else if len(parts) == 1 {
		parsed.Album = parts[0]
	}

	return parsed
}

// =============================================================================
// parseBook - Extract book title and author
// =============================================================================
// Parsing strategy:
//   1. Look for " - " separator between title and author
//   2. If no separator, entire filename is the title
//
// Examples:
//   "Book Title - Author Name.epub" -> Title: "Book Title", Author: "Author Name"
//   "Just The Title.mobi" -> Title: "Just The Title"
// =============================================================================

func parseBook(filename string) *ParsedMedia {
	parsed := &ParsedMedia{
		Kind: "book",
	}

	// Try to split title and author
	if matches := bookAuthorPattern.FindStringSubmatch(filename); len(matches) == 3 {
		parsed.Title = normalizeTitle(matches[1])
		parsed.Author = normalizeTitle(matches[2])
	} else {
		parsed.Title = normalizeTitle(filename)
	}

	return parsed
}

// =============================================================================
// Helper Functions
// =============================================================================

// cleanFilename removes common quality tags and release group info
func cleanFilename(filename string) string {
	lower := strings.ToLower(filename)

	// Remove common quality/release tags
	for _, word := range stripWords {
		lower = strings.ReplaceAll(lower, word, "")
		lower = strings.ReplaceAll(lower, strings.ToUpper(word), "")
	}

	// Remove content in square brackets (usually release group)
	lower = regexp.MustCompile(`\[.*?\]`).ReplaceAllString(lower, "")

	return strings.TrimSpace(lower)
}

// normalizeTitle converts dots/underscores to spaces and title-cases the result
func normalizeTitle(title string) string {
	// Replace dots and underscores with spaces
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Remove multiple spaces
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	// Trim and title case
	title = strings.TrimSpace(title)

	// Basic title casing
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// getShowNameFromPath tries to extract the show name from parent directories
func getShowNameFromPath(dir string) string {
	parts := strings.Split(dir, string(filepath.Separator))

	// Look backwards for a non-season directory
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if !strings.Contains(strings.ToLower(part), "season") {
			return normalizeTitle(part)
		}
	}

	return ""
}

// extractSeasonFromPath tries to extract season number from directory path
func extractSeasonFromPath(dir string) int {
	seasonPattern := regexp.MustCompile(`[Ss]eason\s*(\d{1,2})`)
	if matches := seasonPattern.FindStringSubmatch(dir); len(matches) > 1 {
		season, _ := strconv.Atoi(matches[1])
		return season
	}
	return 0
}

// IsSupportedMediaFile checks if the file extension is supported
func IsSupportedMediaFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return videoExtensions[ext] || audioExtensions[ext] || bookExtensions[ext]
}
