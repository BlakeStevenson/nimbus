package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// ImportConfig holds all configuration for media importing
type ImportConfig struct {
	// Movie naming
	MovieNamingFormat string
	MovieFolderFormat string
	CreateMovieFolder bool
	RenameMovies      bool

	// TV naming
	TVNamingFormat       string
	TVFolderFormat       string
	TVSeasonFolderFormat string
	TVUseSeasonFolders   bool
	CreateSeriesFolder   bool
	RenameEpisodes       bool

	// File management
	ReplaceIllegalCharacters bool
	ColonReplacement         string // "delete", "dash", "space", "spacedash"

	// Quality
	PreferredQuality      string
	EnableQualityUpgrades bool
	UpgradeUntilQuality   string

	// Download client
	CompletedDownloadHandling bool
	RemoveCompletedDownloads  bool

	// Importing
	SkipFreeSpaceCheck  bool
	MinimumFreeSpaceMB  int
	UseHardlinks        bool
	ImportExtraFiles    bool
	ExtraFileExtensions string

	// Advanced
	SetPermissions    bool
	ChmodFolder       string
	ChmodFile         string
	RecycleBinPath    string
	RecycleBinCleanup int
}

// loadConfig loads the import configuration from the config store
func (s *Service) loadConfig(ctx context.Context) (*ImportConfig, error) {
	config := &ImportConfig{
		// Defaults
		MovieNamingFormat:         "{Movie Title} ({Release Year})",
		MovieFolderFormat:         "{Movie Title} ({Release Year})",
		CreateMovieFolder:         true,
		RenameMovies:              true,
		TVNamingFormat:            "{Series Title} - S{season:00}E{episode:00} - {Episode Title}",
		TVFolderFormat:            "{Series Title}",
		TVSeasonFolderFormat:      "Season {season:00}",
		TVUseSeasonFolders:        true,
		CreateSeriesFolder:        true,
		RenameEpisodes:            true,
		ReplaceIllegalCharacters:  true,
		ColonReplacement:          "dash",
		PreferredQuality:          "1080p",
		EnableQualityUpgrades:     true,
		UpgradeUntilQuality:       "1080p",
		CompletedDownloadHandling: true,
		RemoveCompletedDownloads:  false,
		SkipFreeSpaceCheck:        false,
		MinimumFreeSpaceMB:        100,
		UseHardlinks:              true,
		ImportExtraFiles:          true,
		ExtraFileExtensions:       "srt,nfo,txt",
		SetPermissions:            false,
		ChmodFolder:               "755",
		ChmodFile:                 "644",
		RecycleBinPath:            "",
		RecycleBinCleanup:         7,
	}

	// Load each config value
	configMap := map[string]interface{}{
		"downloads.movie_naming_format":         &config.MovieNamingFormat,
		"downloads.movie_folder_format":         &config.MovieFolderFormat,
		"downloads.create_movie_folder":         &config.CreateMovieFolder,
		"downloads.rename_movies":               &config.RenameMovies,
		"downloads.tv_naming_format":            &config.TVNamingFormat,
		"downloads.tv_folder_format":            &config.TVFolderFormat,
		"downloads.tv_season_folder_format":     &config.TVSeasonFolderFormat,
		"downloads.tv_use_season_folders":       &config.TVUseSeasonFolders,
		"downloads.create_series_folder":        &config.CreateSeriesFolder,
		"downloads.rename_episodes":             &config.RenameEpisodes,
		"downloads.replace_illegal_characters":  &config.ReplaceIllegalCharacters,
		"downloads.colon_replacement":           &config.ColonReplacement,
		"downloads.preferred_quality":           &config.PreferredQuality,
		"downloads.enable_quality_upgrades":     &config.EnableQualityUpgrades,
		"downloads.upgrade_until_quality":       &config.UpgradeUntilQuality,
		"downloads.completed_download_handling": &config.CompletedDownloadHandling,
		"downloads.remove_completed_downloads":  &config.RemoveCompletedDownloads,
		"downloads.skip_free_space_check":       &config.SkipFreeSpaceCheck,
		"downloads.minimum_free_space":          &config.MinimumFreeSpaceMB,
		"downloads.use_hardlinks":               &config.UseHardlinks,
		"downloads.import_extra_files":          &config.ImportExtraFiles,
		"downloads.extra_file_extensions":       &config.ExtraFileExtensions,
		"downloads.set_permissions":             &config.SetPermissions,
		"downloads.chmod_folder":                &config.ChmodFolder,
		"downloads.chmod_file":                  &config.ChmodFile,
		"downloads.recycle_bin":                 &config.RecycleBinPath,
		"downloads.recycle_bin_cleanup_days":    &config.RecycleBinCleanup,
	}

	for key, target := range configMap {
		value, err := s.configStore.Get(ctx, key)
		if err != nil {
			// Key doesn't exist, use default
			continue
		}

		// Unmarshal based on target type
		switch v := target.(type) {
		case *string:
			var str string
			if err := json.Unmarshal(value, &str); err == nil {
				*v = str
			}
		case *bool:
			var b bool
			if err := json.Unmarshal(value, &b); err == nil {
				*v = b
			}
		case *int:
			var i int
			// Try int first
			if err := json.Unmarshal(value, &i); err == nil {
				*v = i
			} else {
				// Try float (JSON numbers)
				var f float64
				if err := json.Unmarshal(value, &f); err == nil {
					*v = int(f)
				}
			}
		}
	}

	// Clean up string values (remove quotes if present)
	config.MovieNamingFormat = cleanConfigString(config.MovieNamingFormat)
	config.MovieFolderFormat = cleanConfigString(config.MovieFolderFormat)
	config.TVNamingFormat = cleanConfigString(config.TVNamingFormat)
	config.TVFolderFormat = cleanConfigString(config.TVFolderFormat)
	config.TVSeasonFolderFormat = cleanConfigString(config.TVSeasonFolderFormat)
	config.ColonReplacement = cleanConfigString(config.ColonReplacement)
	config.PreferredQuality = cleanConfigString(config.PreferredQuality)
	config.UpgradeUntilQuality = cleanConfigString(config.UpgradeUntilQuality)
	config.ExtraFileExtensions = cleanConfigString(config.ExtraFileExtensions)
	config.ChmodFolder = cleanConfigString(config.ChmodFolder)
	config.ChmodFile = cleanConfigString(config.ChmodFile)
	config.RecycleBinPath = cleanConfigString(config.RecycleBinPath)

	s.logger.Debug("loaded import configuration",
		zap.String("movie_format", config.MovieNamingFormat),
		zap.String("tv_format", config.TVNamingFormat),
		zap.Bool("use_hardlinks", config.UseHardlinks))

	return config, nil
}

// cleanConfigString removes surrounding quotes from JSON string values
func cleanConfigString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// ValidateConfig validates the configuration
func (c *ImportConfig) Validate() error {
	if c.MovieNamingFormat == "" {
		return fmt.Errorf("movie naming format cannot be empty")
	}
	if c.TVNamingFormat == "" {
		return fmt.Errorf("TV naming format cannot be empty")
	}
	if c.MinimumFreeSpaceMB < 0 {
		return fmt.Errorf("minimum free space cannot be negative")
	}
	if c.RecycleBinCleanup < 0 {
		return fmt.Errorf("recycle bin cleanup days cannot be negative")
	}

	validColonReplacements := []string{"delete", "dash", "space", "spacedash"}
	valid := false
	for _, v := range validColonReplacements {
		if c.ColonReplacement == v {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid colon replacement: %s", c.ColonReplacement)
	}

	return nil
}
