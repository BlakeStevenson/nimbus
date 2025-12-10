package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/blakestevenson/nimbus/internal/library"
	"go.uber.org/zap"
)

// Service handles importing downloaded media into the library
type Service struct {
	queries     *generated.Queries
	configStore *configstore.Store
	logger      *zap.Logger
}

// NewService creates a new importer service
func NewService(queries *generated.Queries, configStore *configstore.Store, logger *zap.Logger) *Service {
	return &Service{
		queries:     queries,
		configStore: configStore,
		logger:      logger.With(zap.String("component", "importer")),
	}
}

// ImportRequest represents a request to import downloaded media
type ImportRequest struct {
	SourcePath   string                 // Path to downloaded file(s)
	MediaType    string                 // "movie" or "tv"
	MediaItemID  *int64                 // Optional: Associated media item ID
	Title        string                 // Media title
	Year         *int                   // Release year (for movies)
	Season       *int                   // Season number (for TV)
	Episode      *int                   // Episode number (for TV)
	EpisodeTitle *string                // Episode title (for TV)
	Quality      *string                // Quality (e.g., "1080p")
	Metadata     map[string]interface{} // Additional metadata
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	Success        bool     `json:"success"`
	FinalPath      string   `json:"final_path"`
	MediaItemID    *int64   `json:"media_item_id,omitempty"`
	Message        string   `json:"message"`
	Error          string   `json:"error,omitempty"`
	CreatedFolders []string `json:"created_folders,omitempty"`
	MovedFiles     []string `json:"moved_files,omitempty"`
	ImportedExtras []string `json:"imported_extras,omitempty"`
}

// Import imports downloaded media into the library
func (s *Service) Import(ctx context.Context, req *ImportRequest) (*ImportResult, error) {
	s.logger.Info("starting media import",
		zap.String("source", req.SourcePath),
		zap.String("type", req.MediaType),
		zap.String("title", req.Title))

	result := &ImportResult{
		CreatedFolders: []string{},
		MovedFiles:     []string{},
		ImportedExtras: []string{},
	}

	// Load configuration
	config, err := s.loadConfig(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("failed to load configuration: %v", err)
		return result, err
	}

	// Determine library path based on media type
	libraryPath, err := s.getLibraryPath(ctx, req.MediaType)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get library path: %v", err)
		return result, err
	}

	// Check if source exists
	if _, err := os.Stat(req.SourcePath); os.IsNotExist(err) {
		result.Error = fmt.Sprintf("source path does not exist: %s", req.SourcePath)
		return result, err
	}

	// Check free space if enabled
	if !config.SkipFreeSpaceCheck {
		if err := s.checkFreeSpace(libraryPath, config.MinimumFreeSpaceMB); err != nil {
			result.Error = fmt.Sprintf("insufficient free space: %v", err)
			return result, err
		}
	}

	// Process based on media type
	var finalPath string
	var mediaItemID *int64

	switch req.MediaType {
	case "movie":
		finalPath, mediaItemID, err = s.importMovie(ctx, req, config, libraryPath, result)
	case "tv", "tv_episode":
		finalPath, mediaItemID, err = s.importTVEpisode(ctx, req, config, libraryPath, result)
	default:
		err = fmt.Errorf("unsupported media type: %s", req.MediaType)
	}

	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	result.Success = true
	result.FinalPath = finalPath
	result.MediaItemID = mediaItemID
	result.Message = fmt.Sprintf("Successfully imported %s to %s", req.Title, finalPath)

	s.logger.Info("media import completed",
		zap.String("title", req.Title),
		zap.String("final_path", finalPath),
		zap.Bool("success", true))

	return result, nil
}

// importMovie imports a movie file
func (s *Service) importMovie(ctx context.Context, req *ImportRequest, config *ImportConfig, libraryPath string, result *ImportResult) (string, *int64, error) {
	// Generate folder name if creating movie folders
	var targetDir string
	if config.CreateMovieFolder {
		folderName := s.applyMovieFolderTemplate(config.MovieFolderFormat, req)
		folderName = s.sanitizePath(folderName, config)
		targetDir = filepath.Join(libraryPath, folderName)

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return "", nil, fmt.Errorf("failed to create movie folder: %w", err)
		}
		result.CreatedFolders = append(result.CreatedFolders, targetDir)
	} else {
		targetDir = libraryPath
	}

	// Generate file name
	fileName := s.applyMovieNamingTemplate(config.MovieNamingFormat, req)
	fileName = s.sanitizePath(fileName, config)

	// Get original extension
	ext := filepath.Ext(req.SourcePath)
	if ext == "" {
		ext = ".mkv" // Default extension
	}

	finalPath := filepath.Join(targetDir, fileName+ext)

	// Move/copy the file
	if config.RenameMovies {
		if err := s.moveFile(req.SourcePath, finalPath, config.UseHardlinks); err != nil {
			return "", nil, fmt.Errorf("failed to move file: %w", err)
		}
		result.MovedFiles = append(result.MovedFiles, finalPath)
	} else {
		// Just move without renaming
		finalPath = filepath.Join(targetDir, filepath.Base(req.SourcePath))
		if err := s.moveFile(req.SourcePath, finalPath, config.UseHardlinks); err != nil {
			return "", nil, fmt.Errorf("failed to move file: %w", err)
		}
		result.MovedFiles = append(result.MovedFiles, finalPath)
	}

	// Import extra files if enabled
	if config.ImportExtraFiles {
		extras := s.findExtraFiles(req.SourcePath, config.ExtraFileExtensions)
		for _, extra := range extras {
			extraName := s.generateExtraFileName(fileName, extra, config)
			extraPath := filepath.Join(targetDir, extraName)
			if err := s.moveFile(extra, extraPath, config.UseHardlinks); err != nil {
				s.logger.Warn("failed to import extra file", zap.String("file", extra), zap.Error(err))
			} else {
				result.ImportedExtras = append(result.ImportedExtras, extraPath)
			}
		}
	}

	// Set permissions if enabled
	if config.SetPermissions {
		s.setPermissions(finalPath, config.ChmodFile)
		s.setPermissions(targetDir, config.ChmodFolder)
	}

	// Upsert/update media item in database
	var mediaItemID *int64
	if req.MediaItemID != nil {
		// Update existing media item with final path
		mediaItemID = req.MediaItemID

		// Create media_files entry for the imported file
		fileSize, _ := s.getFileSize(finalPath)
		_, err := s.queries.CreateMediaFile(ctx, generated.CreateMediaFileParams{
			MediaItemID: req.MediaItemID,
			Path:        finalPath,
			Size:        &fileSize,
			Hash:        nil, // TODO: Calculate hash if needed
		})
		if err != nil {
			s.logger.Warn("failed to create media_files entry",
				zap.String("path", finalPath),
				zap.Error(err))
		} else {
			s.logger.Info("created media_files entry",
				zap.String("path", finalPath),
				zap.Int64("media_item_id", *req.MediaItemID))
		}
	} else {
		// Create new media item via library service
		fileSize, _ := s.getFileSize(finalPath)
		parsed := &library.ParsedMedia{
			Kind:  "movie",
			Title: req.Title,
		}
		if req.Year != nil {
			parsed.Year = *req.Year
		}

		libraryService := library.NewService(s.queries, s.logger)
		itemID, _, err := libraryService.UpsertMovie(ctx, parsed, finalPath, fileSize)
		if err != nil {
			s.logger.Warn("failed to create media item", zap.Error(err))
		} else {
			mediaItemID = &itemID
		}
	}

	return finalPath, mediaItemID, nil
}

// importTVEpisode imports a TV episode file
func (s *Service) importTVEpisode(ctx context.Context, req *ImportRequest, config *ImportConfig, libraryPath string, result *ImportResult) (string, *int64, error) {
	if req.Season == nil || req.Episode == nil {
		return "", nil, fmt.Errorf("season and episode numbers are required for TV imports")
	}

	// Generate series folder
	var seriesFolderName string
	if config.CreateSeriesFolder {
		seriesFolderName = s.applyTVSeriesFolderTemplate(config.TVFolderFormat, req)
		seriesFolderName = s.sanitizePath(seriesFolderName, config)
	} else {
		seriesFolderName = s.sanitizePath(req.Title, config)
	}

	seriesDir := filepath.Join(libraryPath, seriesFolderName)
	if err := os.MkdirAll(seriesDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create series folder: %w", err)
	}
	result.CreatedFolders = append(result.CreatedFolders, seriesDir)

	// Generate season folder if enabled
	var targetDir string
	if config.TVUseSeasonFolders {
		seasonFolderName := s.applyTVSeasonFolderTemplate(config.TVSeasonFolderFormat, req)
		seasonFolderName = s.sanitizePath(seasonFolderName, config)
		targetDir = filepath.Join(seriesDir, seasonFolderName)

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return "", nil, fmt.Errorf("failed to create season folder: %w", err)
		}
		result.CreatedFolders = append(result.CreatedFolders, targetDir)
	} else {
		targetDir = seriesDir
	}

	// Generate episode file name
	fileName := s.applyTVNamingTemplate(config.TVNamingFormat, req)
	fileName = s.sanitizePath(fileName, config)

	// Get original extension
	ext := filepath.Ext(req.SourcePath)
	if ext == "" {
		ext = ".mkv"
	}

	finalPath := filepath.Join(targetDir, fileName+ext)

	// Move/copy the file
	if config.RenameEpisodes {
		if err := s.moveFile(req.SourcePath, finalPath, config.UseHardlinks); err != nil {
			return "", nil, fmt.Errorf("failed to move file: %w", err)
		}
		result.MovedFiles = append(result.MovedFiles, finalPath)
	} else {
		finalPath = filepath.Join(targetDir, filepath.Base(req.SourcePath))
		if err := s.moveFile(req.SourcePath, finalPath, config.UseHardlinks); err != nil {
			return "", nil, fmt.Errorf("failed to move file: %w", err)
		}
		result.MovedFiles = append(result.MovedFiles, finalPath)
	}

	// Import extra files
	if config.ImportExtraFiles {
		extras := s.findExtraFiles(req.SourcePath, config.ExtraFileExtensions)
		for _, extra := range extras {
			extraName := s.generateExtraFileName(fileName, extra, config)
			extraPath := filepath.Join(targetDir, extraName)
			if err := s.moveFile(extra, extraPath, config.UseHardlinks); err != nil {
				s.logger.Warn("failed to import extra file", zap.String("file", extra), zap.Error(err))
			} else {
				result.ImportedExtras = append(result.ImportedExtras, extraPath)
			}
		}
	}

	// Set permissions
	if config.SetPermissions {
		s.setPermissions(finalPath, config.ChmodFile)
		s.setPermissions(targetDir, config.ChmodFolder)
	}

	// Upsert/update media item
	var mediaItemID *int64
	if req.MediaItemID != nil {
		mediaItemID = req.MediaItemID

		// Create media_files entry for the imported file
		fileSize, _ := s.getFileSize(finalPath)
		_, err := s.queries.CreateMediaFile(ctx, generated.CreateMediaFileParams{
			MediaItemID: req.MediaItemID,
			Path:        finalPath,
			Size:        &fileSize,
			Hash:        nil, // TODO: Calculate hash if needed
		})
		if err != nil {
			s.logger.Warn("failed to create media_files entry",
				zap.String("path", finalPath),
				zap.Error(err))
		} else {
			s.logger.Info("created media_files entry",
				zap.String("path", finalPath),
				zap.Int64("media_item_id", *req.MediaItemID))
		}
	} else {
		fileSize, _ := s.getFileSize(finalPath)
		parsed := &library.ParsedMedia{
			Kind:  "tv_episode",
			Title: req.Title,
		}
		if req.Season != nil {
			parsed.Season = *req.Season
		}
		if req.Episode != nil {
			parsed.Episode = *req.Episode
		}

		libraryService := library.NewService(s.queries, s.logger)
		itemID, _, err := libraryService.UpsertTVEpisode(ctx, parsed, finalPath, fileSize)
		if err != nil {
			s.logger.Warn("failed to create media item", zap.Error(err))
		} else {
			mediaItemID = &itemID
		}
	}

	return finalPath, mediaItemID, nil
}

// Helper methods for template application will be in naming.go
// Helper methods for file operations will be in fileops.go
// Configuration loading will be in config.go

func (s *Service) getLibraryPath(ctx context.Context, mediaType string) (string, error) {
	var configKey string
	switch mediaType {
	case "movie":
		configKey = "library.movie_path"
	case "tv", "tv_episode":
		configKey = "library.tv_path"
	case "music":
		configKey = "library.music_path"
	case "book":
		configKey = "library.book_path"
	default:
		configKey = "library.root_path"
	}

	pathValue, err := s.configStore.Get(ctx, configKey)
	if err != nil {
		// Fall back to root path
		pathValue, err = s.configStore.Get(ctx, "library.root_path")
		if err != nil {
			return "/media", nil // Ultimate fallback
		}
	}

	var path string
	if err := json.Unmarshal(pathValue, &path); err != nil {
		return "/media", nil
	}

	return path, nil
}

func (s *Service) sanitizePath(name string, config *ImportConfig) string {
	// Replace illegal characters
	if config.ReplaceIllegalCharacters {
		// Common illegal characters across filesystems
		illegal := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
		for _, char := range illegal {
			if char == ":" {
				// Use colon replacement strategy
				switch config.ColonReplacement {
				case "delete":
					name = strings.ReplaceAll(name, ":", "")
				case "dash":
					name = strings.ReplaceAll(name, ":", "-")
				case "space":
					name = strings.ReplaceAll(name, ":", " ")
				case "spacedash":
					name = strings.ReplaceAll(name, ":", " - ")
				}
			} else {
				name = strings.ReplaceAll(name, char, "")
			}
		}
	}

	// Trim whitespace and dots
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")

	return name
}

func (s *Service) moveFile(src, dst string, useHardlinks bool) error {
	// Check if src and dst are on the same filesystem for hardlinks
	if useHardlinks {
		// Try hardlink first
		if err := os.Link(src, dst); err == nil {
			// Hardlink successful, remove source
			os.Remove(src)
			return nil
		} else {
			// Hardlink failed, fall through to copy
			s.logger.Debug("hardlink failed, falling back to copy", zap.Error(err))
		}
	}

	// Copy file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err == nil {
		os.Chmod(dst, srcInfo.Mode())
	}

	// Remove source
	return os.Remove(src)
}

func (s *Service) findExtraFiles(mainFile string, extensions string) []string {
	dir := filepath.Dir(mainFile)
	baseName := strings.TrimSuffix(filepath.Base(mainFile), filepath.Ext(mainFile))

	extList := strings.Split(extensions, ",")
	var extras []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return extras
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		for _, ext := range extList {
			ext = strings.TrimSpace(ext)
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}

			// Check if file matches base name and has the extension
			if strings.HasPrefix(name, baseName) && strings.HasSuffix(name, ext) {
				extras = append(extras, filepath.Join(dir, name))
			}
		}
	}

	return extras
}

func (s *Service) generateExtraFileName(baseName, extraFile string, config *ImportConfig) string {
	ext := filepath.Ext(extraFile)

	// Check if extra file has a suffix (e.g., .en.srt, .forced.srt)
	extraBase := strings.TrimSuffix(filepath.Base(extraFile), ext)
	mainBase := baseName

	suffix := strings.TrimPrefix(extraBase, strings.TrimSuffix(mainBase, filepath.Ext(mainBase)))

	return baseName + suffix + ext
}

func (s *Service) checkFreeSpace(path string, minFreeMB int) error {
	// Platform-specific implementation would go here
	// For now, we'll skip the check
	return nil
}

func (s *Service) setPermissions(path string, mode string) {
	// Convert octal string to mode
	var perm os.FileMode
	fmt.Sscanf(mode, "%o", &perm)
	os.Chmod(path, perm)
}

func (s *Service) getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// applyMovieNamingTemplate applies naming template with token replacement
func (s *Service) applyMovieNamingTemplate(template string, req *ImportRequest) string {
	result := template

	// Replace tokens
	result = strings.ReplaceAll(result, "{Movie Title}", req.Title)

	if req.Year != nil {
		result = strings.ReplaceAll(result, "{Release Year}", fmt.Sprintf("%d", *req.Year))
	} else {
		result = strings.ReplaceAll(result, "{Release Year}", "")
	}

	if req.Quality != nil {
		result = strings.ReplaceAll(result, "{Quality}", *req.Quality)
	} else {
		result = strings.ReplaceAll(result, "{Quality}", "")
	}

	// Clean up double spaces and brackets
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = regexp.MustCompile(`\[\s*\]`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`\(\s*\)`).ReplaceAllString(result, "")

	return strings.TrimSpace(result)
}

func (s *Service) applyMovieFolderTemplate(template string, req *ImportRequest) string {
	return s.applyMovieNamingTemplate(template, req)
}

func (s *Service) applyTVNamingTemplate(template string, req *ImportRequest) string {
	result := template

	result = strings.ReplaceAll(result, "{Series Title}", req.Title)

	if req.Season != nil {
		result = strings.ReplaceAll(result, "{Season}", fmt.Sprintf("%d", *req.Season))
		result = regexp.MustCompile(`\{season:(\d+)\}`).ReplaceAllStringFunc(result, func(m string) string {
			re := regexp.MustCompile(`\{season:(\d+)\}`)
			matches := re.FindStringSubmatch(m)
			if len(matches) > 1 {
				format := fmt.Sprintf("%%0%sd", matches[1])
				return fmt.Sprintf(format, *req.Season)
			}
			return m
		})
	}

	if req.Episode != nil {
		result = strings.ReplaceAll(result, "{Episode}", fmt.Sprintf("%d", *req.Episode))
		result = regexp.MustCompile(`\{episode:(\d+)\}`).ReplaceAllStringFunc(result, func(m string) string {
			re := regexp.MustCompile(`\{episode:(\d+)\}`)
			matches := re.FindStringSubmatch(m)
			if len(matches) > 1 {
				format := fmt.Sprintf("%%0%sd", matches[1])
				return fmt.Sprintf(format, *req.Episode)
			}
			return m
		})
	}

	if req.EpisodeTitle != nil {
		result = strings.ReplaceAll(result, "{Episode Title}", *req.EpisodeTitle)
	} else {
		result = strings.ReplaceAll(result, "{Episode Title}", "")
	}

	if req.Quality != nil {
		result = strings.ReplaceAll(result, "{Quality}", *req.Quality)
	} else {
		result = strings.ReplaceAll(result, "{Quality}", "")
	}

	// Clean up
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = regexp.MustCompile(`\[\s*\]`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`\(\s*\)`).ReplaceAllString(result, "")
	result = strings.ReplaceAll(result, " - -", " -")

	return strings.TrimSpace(result)
}

func (s *Service) applyTVSeriesFolderTemplate(template string, req *ImportRequest) string {
	result := template
	result = strings.ReplaceAll(result, "{Series Title}", req.Title)

	if req.Year != nil {
		result = strings.ReplaceAll(result, "{Year}", fmt.Sprintf("%d", *req.Year))
	} else {
		result = strings.ReplaceAll(result, "{Year}", "")
	}

	return strings.TrimSpace(result)
}

func (s *Service) applyTVSeasonFolderTemplate(template string, req *ImportRequest) string {
	result := template

	if req.Season != nil {
		result = strings.ReplaceAll(result, "{Season}", fmt.Sprintf("%d", *req.Season))
		result = regexp.MustCompile(`\{season:(\d+)\}`).ReplaceAllStringFunc(result, func(m string) string {
			re := regexp.MustCompile(`\{season:(\d+)\}`)
			matches := re.FindStringSubmatch(m)
			if len(matches) > 1 {
				format := fmt.Sprintf("%%0%sd", matches[1])
				return fmt.Sprintf(format, *req.Season)
			}
			return m
		})
	}

	return strings.TrimSpace(result)
}
