-- Add comprehensive download configuration options (Sonarr-like)
-- These settings control how downloaded media is organized and named

-- ============================================================================
-- Media Management - File Naming
-- ============================================================================

INSERT INTO config (key, value, metadata) VALUES
    -- Movies
    ('downloads.movie_naming_format', '"{Movie Title} ({Release Year})"', jsonb_build_object(
        'title', 'Movie File Naming Format',
        'description', 'Template for naming movie files. Available tokens: {Movie Title}, {Release Year}, {Quality}, {Edition}, {IMDb ID}',
        'type', 'text',
        'category', 'downloads',
        'section', 'Movie Naming'
    )),
    ('downloads.movie_folder_format', '"{Movie Title} ({Release Year})"', jsonb_build_object(
        'title', 'Movie Folder Format',
        'description', 'Template for movie folder names. Available tokens: {Movie Title}, {Release Year}, {IMDb ID}',
        'type', 'text',
        'category', 'downloads',
        'section', 'Movie Naming'
    )),

    -- TV Shows
    ('downloads.tv_naming_format', '"{Series Title} - S{season:00}E{episode:00} - {Episode Title}"', jsonb_build_object(
        'title', 'TV Episode Naming Format',
        'description', 'Template for TV episode files. Available tokens: {Series Title}, {Season}, {Episode}, {Episode Title}, {Quality}, {Release Date}',
        'type', 'text',
        'category', 'downloads',
        'section', 'TV Naming'
    )),
    ('downloads.tv_folder_format', '"{Series Title}"', jsonb_build_object(
        'title', 'TV Series Folder Format',
        'description', 'Template for TV series folder names. Available tokens: {Series Title}, {Year}, {TVDb ID}',
        'type', 'text',
        'category', 'downloads',
        'section', 'TV Naming'
    )),
    ('downloads.tv_season_folder_format', '"Season {season:00}"', jsonb_build_object(
        'title', 'TV Season Folder Format',
        'description', 'Template for season folder names within series folder. Available tokens: {Season}',
        'type', 'text',
        'category', 'downloads',
        'section', 'TV Naming'
    )),
    ('downloads.tv_use_season_folders', 'true', jsonb_build_object(
        'title', 'Use Season Folders',
        'description', 'Create separate folders for each season within the series folder',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'TV Naming'
    )),

-- ============================================================================
-- Media Management - Folder Structure
-- ============================================================================

    ('downloads.create_series_folder', 'true', jsonb_build_object(
        'title', 'Create Series Folder',
        'description', 'Automatically create a folder for each TV series',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Folder Structure'
    )),
    ('downloads.create_movie_folder', 'true', jsonb_build_object(
        'title', 'Create Movie Folder',
        'description', 'Automatically create a folder for each movie',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Folder Structure'
    )),

-- ============================================================================
-- Media Management - File Management
-- ============================================================================

    ('downloads.rename_episodes', 'true', jsonb_build_object(
        'title', 'Rename Episodes',
        'description', 'Automatically rename downloaded episodes to match naming format',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'File Management'
    )),
    ('downloads.rename_movies', 'true', jsonb_build_object(
        'title', 'Rename Movies',
        'description', 'Automatically rename downloaded movies to match naming format',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'File Management'
    )),
    ('downloads.replace_illegal_characters', 'true', jsonb_build_object(
        'title', 'Replace Illegal Characters',
        'description', 'Replace characters that are illegal in filenames with legal alternatives',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'File Management'
    )),
    ('downloads.colon_replacement', '"dash"', jsonb_build_object(
        'title', 'Colon Replacement',
        'description', 'How to handle colons in filenames',
        'type', 'select',
        'values', jsonb_build_array('delete', 'dash', 'space', 'spacedash'),
        'category', 'downloads',
        'section', 'File Management'
    )),

-- ============================================================================
-- Media Management - Quality & Upgrades
-- ============================================================================

    ('downloads.quality_profiles', '["Any", "SD", "720p", "1080p", "2160p"]', jsonb_build_object(
        'title', 'Quality Profiles',
        'description', 'Available quality profiles for downloads (ordered by preference)',
        'type', 'array',
        'category', 'downloads',
        'section', 'Quality'
    )),
    ('downloads.preferred_quality', '"1080p"', jsonb_build_object(
        'title', 'Preferred Quality',
        'description', 'Preferred quality for downloads',
        'type', 'select',
        'values', jsonb_build_array('Any', 'SD', '720p', '1080p', '2160p'),
        'category', 'downloads',
        'section', 'Quality'
    )),
    ('downloads.enable_quality_upgrades', 'true', jsonb_build_object(
        'title', 'Enable Quality Upgrades',
        'description', 'Automatically upgrade to higher quality releases when available',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Quality'
    )),
    ('downloads.upgrade_until_quality', '"1080p"', jsonb_build_object(
        'title', 'Upgrade Until Quality',
        'description', 'Stop upgrading once this quality is reached',
        'type', 'select',
        'values', jsonb_build_array('SD', '720p', '1080p', '2160p'),
        'category', 'downloads',
        'section', 'Quality'
    )),

-- ============================================================================
-- Download Client Settings
-- ============================================================================

    ('downloads.completed_download_handling', 'true', jsonb_build_object(
        'title', 'Enable Completed Download Handling',
        'description', 'Automatically import completed downloads and move to library',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Download Client'
    )),
    ('downloads.remove_completed_downloads', 'false', jsonb_build_object(
        'title', 'Remove Completed Downloads',
        'description', 'Remove downloads from download client after import',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Download Client'
    )),
    ('downloads.check_for_finished_download_interval', '1', jsonb_build_object(
        'title', 'Check For Finished Downloads Interval',
        'description', 'Interval in minutes to check for completed downloads',
        'type', 'number',
        'category', 'downloads',
        'section', 'Download Client'
    )),

-- ============================================================================
-- Importing Settings
-- ============================================================================

    ('downloads.skip_free_space_check', 'false', jsonb_build_object(
        'title', 'Skip Free Space Check',
        'description', 'Skip checking free space before importing',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Importing'
    )),
    ('downloads.minimum_free_space', '100', jsonb_build_object(
        'title', 'Minimum Free Space (MB)',
        'description', 'Minimum free space required before importing (in megabytes)',
        'type', 'number',
        'category', 'downloads',
        'section', 'Importing'
    )),
    ('downloads.use_hardlinks', 'true', jsonb_build_object(
        'title', 'Use Hardlinks Instead of Copy',
        'description', 'Use hardlinks when possible to save disk space (requires same filesystem)',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Importing'
    )),
    ('downloads.import_extra_files', 'true', jsonb_build_object(
        'title', 'Import Extra Files',
        'description', 'Import extra files found with media (subtitles, nfo, etc)',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Importing'
    )),
    ('downloads.extra_file_extensions', '"srt,nfo,txt"', jsonb_build_object(
        'title', 'Extra File Extensions',
        'description', 'Comma-separated list of extra file extensions to import',
        'type', 'text',
        'category', 'downloads',
        'section', 'Importing'
    )),

-- ============================================================================
-- Advanced Settings
-- ============================================================================

    ('downloads.set_permissions', 'false', jsonb_build_object(
        'title', 'Set Permissions',
        'description', 'Set permissions on imported files and folders',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Advanced'
    )),
    ('downloads.chmod_folder', '"755"', jsonb_build_object(
        'title', 'Folder Permissions',
        'description', 'Permissions for created folders (octal format)',
        'type', 'text',
        'category', 'downloads',
        'section', 'Advanced'
    )),
    ('downloads.chmod_file', '"644"', jsonb_build_object(
        'title', 'File Permissions',
        'description', 'Permissions for imported files (octal format)',
        'type', 'text',
        'category', 'downloads',
        'section', 'Advanced'
    )),
    ('downloads.recycle_bin', '""', jsonb_build_object(
        'title', 'Recycle Bin Path',
        'description', 'Move deleted files to this location instead of permanently deleting (empty = disabled)',
        'type', 'text',
        'category', 'downloads',
        'section', 'Advanced'
    )),
    ('downloads.recycle_bin_cleanup_days', '7', jsonb_build_object(
        'title', 'Recycle Bin Cleanup Days',
        'description', 'Days to keep files in recycle bin before permanent deletion (0 = keep forever)',
        'type', 'number',
        'category', 'downloads',
        'section', 'Advanced'
    ))

ON CONFLICT (key) DO NOTHING;
