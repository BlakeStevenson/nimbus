package library

import (
	"io/fs"
	"os"
	"path/filepath"
)

// =============================================================================
// WalkMediaFiles - Recursively traverse a directory and collect media files
// =============================================================================
// This function walks a root directory and returns all file paths that match
// supported media extensions (video, audio, books).
//
// Features:
//   - Recursive directory traversal using filepath.WalkDir
//   - Filters files by extension using IsSupportedMediaFile
//   - Skips hidden files and directories (starting with .)
//   - Skips common system directories (@eaDir, .thumbnails, etc.)
//   - Returns absolute paths for all matching files
//
// Parameters:
//   - root: The root directory path to start scanning from
//
// Returns:
//   - []string: List of absolute file paths for all discovered media files
//   - error: Any error encountered during directory traversal
//
// Usage:
//   files, err := WalkMediaFiles("/media/library")
//   if err != nil {
//       log.Fatal(err)
//   }
//   fmt.Printf("Found %d media files\n", len(files))
// =============================================================================

func WalkMediaFiles(root string) ([]string, error) {
	var mediaFiles []string

	// Verify root directory exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, err
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Log the error but continue walking
			// This prevents permission errors from stopping the entire scan
			return nil
		}

		// Skip hidden files and directories
		if d.Name()[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common system/metadata directories
		if d.IsDir() && shouldSkipDirectory(d.Name()) {
			return filepath.SkipDir
		}

		// Only process regular files
		if !d.IsDir() {
			// Check if this is a supported media file
			if IsSupportedMediaFile(path) {
				// Get absolute path
				absPath, err := filepath.Abs(path)
				if err != nil {
					absPath = path
				}
				mediaFiles = append(mediaFiles, absPath)
			}
		}

		return nil
	})

	return mediaFiles, err
}

// =============================================================================
// shouldSkipDirectory - Check if a directory should be skipped during scan
// =============================================================================
// Returns true for directories that typically contain metadata, thumbnails,
// or other non-media content that should not be scanned.
//
// Skipped directories:
//   - @eaDir: Synology metadata directory
//   - .thumbnails: Thumbnail cache
//   - .AppleDouble: macOS resource fork directory
//   - Thumbs.db: Windows thumbnail cache
//   - .DS_Store: macOS folder metadata
//   - $RECYCLE.BIN: Windows recycle bin
//   - System Volume Information: Windows system folder
//   - lost+found: Linux filesystem recovery directory
// =============================================================================

func shouldSkipDirectory(name string) bool {
	skipDirs := map[string]bool{
		"@eaDir":                    true, // Synology
		".thumbnails":               true,
		".AppleDouble":              true,
		"Thumbs.db":                 true,
		".DS_Store":                 true,
		"$RECYCLE.BIN":              true,
		"System Volume Information": true,
		"lost+found":                true,
		".Trash":                    true,
		".cache":                    true,
		".tmp":                      true,
		"@Recycle":                  true, // QNAP
		"#recycle":                  true, // QNAP
		".TemporaryItems":           true,
		".Spotlight-V100":           true, // macOS Spotlight index
		".fseventsd":                true, // macOS filesystem events
	}

	return skipDirs[name]
}

// =============================================================================
// WalkMediaFilesChan - Channel-based version for streaming results
// =============================================================================
// This alternative version streams file paths through a channel as they are
// discovered, which is more memory-efficient for very large libraries.
//
// The channel is closed when the walk is complete or an error occurs.
//
// Usage:
//   fileChan := make(chan string, 100)
//   errChan := make(chan error, 1)
//
//   go WalkMediaFilesChan("/media/library", fileChan, errChan)
//
//   for {
//       select {
//       case path, ok := <-fileChan:
//           if !ok {
//               return // Channel closed, scan complete
//           }
//           // Process path
//       case err := <-errChan:
//           // Handle error
//       }
//   }
// =============================================================================

func WalkMediaFilesChan(root string, fileChan chan<- string, errChan chan<- error) {
	defer close(fileChan)

	// Verify root directory exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		errChan <- err
		return
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Don't stop on permission errors, just skip
			return nil
		}

		// Skip hidden files and directories
		if d.Name()[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip system directories
		if d.IsDir() && shouldSkipDirectory(d.Name()) {
			return filepath.SkipDir
		}

		// Process regular files
		if !d.IsDir() && IsSupportedMediaFile(path) {
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}

			select {
			case fileChan <- absPath:
			default:
				// Channel full, this shouldn't happen with proper buffer size
				// but we don't want to block indefinitely
			}
		}

		return nil
	})

	if err != nil {
		errChan <- err
	}
}

// =============================================================================
// GetFileInfo - Extract basic file information
// =============================================================================
// Returns file size and other metadata needed for the media_files table.
//
// Returns:
//   - size: File size in bytes
//   - err: Any error reading file info
// =============================================================================

func GetFileInfo(path string) (size int64, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// =============================================================================
// CountMediaFiles - Count media files without storing paths
// =============================================================================
// More memory-efficient than WalkMediaFiles when you only need a count.
//
// Returns:
//   - count: Number of media files found
//   - err: Any error during traversal
// =============================================================================

func CountMediaFiles(root string) (int, error) {
	count := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.Name()[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() && shouldSkipDirectory(d.Name()) {
			return filepath.SkipDir
		}

		if !d.IsDir() && IsSupportedMediaFile(path) {
			count++
		}

		return nil
	})

	return count, err
}
