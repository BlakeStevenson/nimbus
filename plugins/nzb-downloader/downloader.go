package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SegmentJob represents a segment download job
type SegmentJob struct {
	FileIndex    int
	SegmentIndex int
	Segment      NZBSegment
	Retries      int
}

// SegmentResult represents the result of a segment download
type SegmentResult struct {
	FileIndex    int
	SegmentIndex int
	Data         []byte
	Error        error
}

// FastDownloader handles efficient parallel downloads
type FastDownloader struct {
	connPool        []*NNTPClient
	jobQueue        chan *SegmentJob
	resultQueue     chan *SegmentResult
	workers         int
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	totalBytes      int64
	downloadedBytes int64
	download        *Download // Reference to download for logging
	activeWorkers   int32     // Track active workers
}

// NewFastDownloader creates a new fast downloader with connection pool
func NewFastDownloader(ctx context.Context, server NNTPServer, download *Download) (*FastDownloader, error) {
	numConnections := server.Connections
	if numConnections <= 0 {
		numConnections = 10
	}

	ctx, cancel := context.WithCancel(ctx)

	// Use larger buffers to prevent blocking
	queueSize := numConnections * 10
	if queueSize < 1000 {
		queueSize = 1000
	}

	fd := &FastDownloader{
		connPool:    make([]*NNTPClient, 0, numConnections),
		jobQueue:    make(chan *SegmentJob, queueSize),
		resultQueue: make(chan *SegmentResult, queueSize),
		workers:     numConnections,
		ctx:         ctx,
		cancel:      cancel,
		download:    download,
	}

	download.AddLog(fmt.Sprintf("Initializing %d connections to %s:%d", numConnections, server.Host, server.Port))

	// Create connection pool in parallel for faster startup
	var connWg sync.WaitGroup
	var connMu sync.Mutex
	successCount := int32(0)

	for i := 0; i < numConnections; i++ {
		connWg.Add(1)
		go func(idx int) {
			defer connWg.Done()

			conn, err := DialNNTP(server.Host, server.Port, server.UseSSL)
			if err != nil {
				download.AddLog(fmt.Sprintf("Connection %d failed to dial: %v", idx, err))
				return
			}

			if err := conn.Authenticate(server.Username, server.Password); err != nil {
				download.AddLog(fmt.Sprintf("Connection %d failed to authenticate: %v", idx, err))
				conn.Close()
				return
			}

			connMu.Lock()
			fd.connPool = append(fd.connPool, conn)
			connMu.Unlock()

			atomic.AddInt32(&successCount, 1)
		}(i)
	}

	connWg.Wait()

	if len(fd.connPool) == 0 {
		download.AddLog("Failed to establish any NNTP connections")
		cancel()
		// Clean up any partial resources
		close(fd.jobQueue)
		close(fd.resultQueue)
		return nil, fmt.Errorf("failed to establish any NNTP connections")
	}

	download.AddLog(fmt.Sprintf("Created %d/%d connections successfully", len(fd.connPool), numConnections))

	// If we have fewer connections than requested, that's okay but log it
	if len(fd.connPool) < numConnections {
		download.AddLog(fmt.Sprintf("WARNING: Only %d/%d connections available, continuing with reduced capacity", len(fd.connPool), numConnections))
	}

	// Start workers (one per connection)
	for i := 0; i < len(fd.connPool); i++ {
		fd.wg.Add(1)
		go fd.worker(i, fd.connPool[i])
	}

	return fd, nil
}

// worker processes download jobs
func (fd *FastDownloader) worker(id int, conn *NNTPClient) {
	defer fd.wg.Done()

	defer func() {
		if r := recover(); r != nil {
			fd.download.AddLog(fmt.Sprintf("PANIC in worker %d: %v", id, r))
		}
	}()

	if id == 0 {
		fd.download.AddLog(fmt.Sprintf("Started %d workers", len(fd.connPool)))
	}
	jobsProcessed := 0

	for {
		select {
		case <-fd.ctx.Done():
			return
		case job := <-fd.jobQueue:
			if job == nil {
				return
			}

			if jobsProcessed == 0 && id == 0 {
				fd.download.AddLog("Workers processing segments")
			}
			jobsProcessed++

			// Validate message ID
			if job.Segment.MessageID == "" {
				fd.resultQueue <- &SegmentResult{
					FileIndex:    job.FileIndex,
					SegmentIndex: job.SegmentIndex,
					Error:        fmt.Errorf("empty message ID"),
				}
				continue
			}

			// Download segment
			article, err := conn.GetArticle(job.Segment.MessageID)
			if err != nil {
				// Retry logic
				if job.Retries < 3 {
					job.Retries++
					select {
					case fd.jobQueue <- job:
					case <-fd.ctx.Done():
						return
					}
					continue
				}

				fd.resultQueue <- &SegmentResult{
					FileIndex:    job.FileIndex,
					SegmentIndex: job.SegmentIndex,
					Error:        err,
				}
				continue
			}

			// Decode yEnc
			decoded, err := DecodeArticle(article)
			if err != nil {
				fd.resultQueue <- &SegmentResult{
					FileIndex:    job.FileIndex,
					SegmentIndex: job.SegmentIndex,
					Error:        fmt.Errorf("failed to decode yEnc: %v", err),
				}
				continue
			}

			// Validate decoded size roughly matches expected size
			// Allow some tolerance for yEnc overhead, but if it's way off, something is wrong
			expectedSize := job.Segment.Bytes
			decodedSize := int64(len(decoded))
			tolerance := int64(float64(expectedSize) * 0.5) // 50% tolerance

			if decodedSize > expectedSize+tolerance || decodedSize < expectedSize-tolerance {
				fd.download.AddLog(fmt.Sprintf("WARNING: Segment size mismatch - expected ~%d bytes, got %d bytes (segment %d of file %d)",
					expectedSize, decodedSize, job.SegmentIndex, job.FileIndex))
			}

			// Send result
			fd.resultQueue <- &SegmentResult{
				FileIndex:    job.FileIndex,
				SegmentIndex: job.SegmentIndex,
				Data:         decoded,
			}

			// Update progress
			atomic.AddInt64(&fd.downloadedBytes, decodedSize)
		}
	}
}

// Download downloads an NZB with all its files
func (fd *FastDownloader) Download(download *Download, downloadDir string) error {
	nzbData := download.NZBData

	fd.download.AddLog(fmt.Sprintf("Starting download process for %d files", len(nzbData.Files)))

	// Calculate total bytes
	fd.totalBytes = download.TotalBytes

	// Create file writers map
	fileWriters := make(map[int]*FileAssembler)

	fd.download.AddLog(fmt.Sprintf("Creating output files in %s", downloadDir))

	for fileIdx, file := range nzbData.Files {
		filename := file.Filename()
		if filename == "" || len(filename) == 0 {
			filename = fmt.Sprintf("%s-part%d.bin", download.Name, fileIdx+1)
		}
		filename = filepath.Base(filename)
		filename = CleanFilename(filename)

		outputPath := filepath.Join(downloadDir, filename)

		assembler, err := NewFileAssembler(outputPath, len(file.Segments))
		if err != nil {
			fd.download.AddLog(fmt.Sprintf("ERROR creating file %s: %v", filename, err))
			return fmt.Errorf("failed to create file assembler: %v", err)
		}
		fileWriters[fileIdx] = assembler
	}

	fd.download.AddLog(fmt.Sprintf("Created %d output files", len(fileWriters)))

	// Count total segments first
	totalSegments := 0
	for _, file := range nzbData.Files {
		totalSegments += len(file.Segments)
	}

	fd.download.AddLog(fmt.Sprintf("Queueing %d segments across %d files (%.2f MB total)",
		totalSegments, len(nzbData.Files), float64(fd.totalBytes)/(1024*1024)))

	// Queue all segment jobs in background to avoid blocking
	go func() {
		for fileIdx, file := range nzbData.Files {
			for _, segment := range file.Segments {
				select {
				case fd.jobQueue <- &SegmentJob{
					FileIndex:    fileIdx,
					SegmentIndex: segment.Number - 1, // Convert from 1-based to 0-based indexing
					Segment:      segment,
					Retries:      0,
				}:
				case <-fd.ctx.Done():
					return
				}
			}
		}
	}()

	fd.download.AddLog("Processing results...")

	// Process results
	receivedSegments := 0
	failedSegments := 0

	startTime := time.Now()
	lastUpdate := time.Now()
	lastBytes := int64(0)

	for receivedSegments+failedSegments < totalSegments {
		select {
		case <-fd.ctx.Done():
			return fmt.Errorf("download cancelled")
		case result := <-fd.resultQueue:
			if result == nil {
				fd.download.AddLog("ERROR: Received nil result")
				failedSegments++
				continue
			}

			if result.Error != nil {
				fd.download.AddLog(fmt.Sprintf("Segment %d/%d failed: %v", result.FileIndex, result.SegmentIndex, result.Error))
				failedSegments++
				continue
			}

			// Write segment to assembler
			assembler, ok := fileWriters[result.FileIndex]
			if !ok {
				fd.download.AddLog(fmt.Sprintf("ERROR: No assembler for file index %d", result.FileIndex))
				failedSegments++
				continue
			}

			if assembler == nil {
				fd.download.AddLog(fmt.Sprintf("ERROR: Assembler is nil for file index %d", result.FileIndex))
				failedSegments++
				continue
			}

			if err := assembler.WriteSegment(result.SegmentIndex, result.Data); err != nil {
				fd.download.AddLog(fmt.Sprintf("Failed to write segment %d/%d: %v", result.FileIndex, result.SegmentIndex, err))
				failedSegments++
				continue
			}

			receivedSegments++

			// Update progress
			downloaded := atomic.LoadInt64(&fd.downloadedBytes)
			download.DownloadedBytes = downloaded
			download.Progress = float64(downloaded) / float64(fd.totalBytes) * 100

			// Calculate speed and ETA every second
			now := time.Now()
			if now.Sub(lastUpdate) >= time.Second {
				elapsed := now.Sub(lastUpdate).Seconds()
				if elapsed > 0 {
					download.Speed = int64(float64(downloaded-lastBytes) / elapsed)

					if download.Speed > 0 {
						remainingBytes := fd.totalBytes - downloaded
						download.ETA = remainingBytes / download.Speed
					}
				}
				lastUpdate = now
				lastBytes = downloaded

				// Progress log every 5 seconds to logs
				if int(now.Sub(startTime).Seconds())%5 == 0 {
					progress := float64(receivedSegments) / float64(totalSegments) * 100
					fd.download.AddLog(fmt.Sprintf("Progress: %d/%d segments (%.1f%%) - %.2f MB/s",
						receivedSegments, totalSegments, progress, float64(download.Speed)/(1024*1024)))
				}
			}
		}
	}

	// Finalize all files
	fd.download.AddLog("Finalizing files...")
	downloadedFiles := []string{}
	for fileIdx, assembler := range fileWriters {
		if err := assembler.Close(); err != nil {
			fd.download.AddLog(fmt.Sprintf("ERROR: Failed to finalize file %d: %v", fileIdx, err))
		} else {
			downloadedFiles = append(downloadedFiles, assembler.filepath)
		}
	}

	if failedSegments > 0 {
		fd.download.AddLog(fmt.Sprintf("WARNING: %d segments failed to download", failedSegments))
		return fmt.Errorf("%d segments failed to download", failedSegments)
	}

	totalTime := time.Since(startTime).Seconds()
	avgSpeed := float64(fd.totalBytes) / totalTime / (1024 * 1024)
	fd.download.AddLog(fmt.Sprintf("Downloaded %d segments in %.1fs (avg %.2f MB/s)",
		totalSegments, totalTime, avgSpeed))

	return nil
}

// parseVolumeFromFilename attempts to extract volume number from filename
// Supports patterns like: .part01.rar, .part001.rar, .r01, .001.rar
// Returns 0-based index (0 for first volume), or -1 if not found
func parseVolumeFromFilename(filename string) int {
	filename = strings.ToLower(filename)

	// Pattern 1: .partXX.rar or .partXXX.rar (most common for obfuscated)
	re := regexp.MustCompile(`\.part(\d+)\.rar$`)
	if matches := re.FindStringSubmatch(filename); len(matches) == 2 {
		var num int
		fmt.Sscanf(matches[1], "%d", &num)
		if num > 0 {
			return num - 1 // Convert to 0-based
		}
	}

	// Pattern 2: .rXX (old RAR naming: .rar, .r00, .r01, .r02...)
	re = regexp.MustCompile(`\.r(\d+)$`)
	if matches := re.FindStringSubmatch(filename); len(matches) == 2 {
		var num int
		fmt.Sscanf(matches[1], "%d", &num)
		return num + 1 // .r00 is second volume (first is .rar)
	}

	// Pattern 3: Check if it ends with .rar (could be first volume in old naming)
	if strings.HasSuffix(filename, ".rar") && !strings.Contains(filename, ".part") {
		// Could be first volume, but only if no other patterns match
		// Return -1 to let other methods determine
		return -1
	}

	return -1 // Could not determine from filename
}

// parseRARVolumeNumber extracts the volume number from a RAR file header
// Returns 0 for first volume, 1 for second volume, etc.
func parseRARVolumeNumber(header []byte, length int) int {
	// RAR 5.0 format: starts with "Rar!\x1a\x07\x01\x00"
	if length >= 8 && string(header[:7]) == "Rar!\x1a\x07\x01" {
		return parseRAR5VolumeNumber(header, length)
	}

	// RAR 4.x format: starts with "Rar!\x1a\x07\x00"
	// For older formats, fall back to checking common naming patterns or assume volume 1
	return parseRAR4VolumeNumber(header, length)
}

// parseRAR5VolumeNumber parses volume number from RAR 5.0 format
func parseRAR5VolumeNumber(header []byte, length int) int {
	// RAR 5.0 header structure:
	// 0-7: Signature "Rar!\x1a\x07\x01\x00"
	// 8-11: Header CRC32 (4 bytes)
	// 12+: Main archive header with vint fields

	if length < 20 {
		return 0 // Not enough data, assume first volume
	}

	pos := 12 // Start after signature (8 bytes) and CRC32 (4 bytes)

	// Parse vint fields in sequence:
	// 1. Header size
	_, bytesRead := readVint(header[pos:])
	pos += bytesRead

	// 2. Header type (should be 1 for main archive header)
	headerType, bytesRead := readVint(header[pos:])
	pos += bytesRead
	if headerType != 1 {
		return 0 // Not a main archive header
	}

	// 3. Header flags
	_, bytesRead = readVint(header[pos:])
	pos += bytesRead

	// 4. Extra area size (if header flags & 0x0001)
	// We'll skip this check for simplicity since extra area is optional

	// 5. Archive flags - this tells us if volume number field exists
	archiveFlags, bytesRead := readVint(header[pos:])
	pos += bytesRead

	// Check if this is a multivolume archive (0x0001) and has volume number field (0x0002)
	if archiveFlags&0x0001 == 0 {
		return 0 // Not a multivolume archive
	}

	if archiveFlags&0x0002 == 0 {
		return 0 // First volume (no volume number field)
	}

	// 6. Volume number field (present for all volumes except first)
	// Value is 1 for second volume, 2 for third, etc.
	if pos >= length {
		return 0 // Not enough data
	}

	volumeNum, _ := readVint(header[pos:])
	// RAR stores 1 for second volume, 2 for third, etc.
	// We return 0-based: 0 for first, 1 for second, etc.
	return int(volumeNum) + 1 // +1 because first volume is 0, but RAR starts counting at 1 for second volume
}

// parseRAR4VolumeNumber attempts to parse volume number from RAR 4.x format
// This is more complex, so we use a simpler heuristic
func parseRAR4VolumeNumber(header []byte, length int) int {
	// RAR 4.x format is more complex and varies
	// For now, check for the NEW_NUMBERING flag at offset 10
	// and try to read volume number if present

	if length < 13 {
		return 0
	}

	// RAR 4.x: Check flags at offset 10-11
	flags := uint16(header[10]) | (uint16(header[11]) << 8)

	// 0x0100 = MHD_VOLUME (archive is part of multivolume set)
	// 0x0001 = MHD_FIRSTVOLUME (first volume)
	if flags&0x0100 == 0 {
		return 0 // Not a multivolume archive
	}

	if flags&0x0001 != 0 {
		return 0 // First volume
	}

	// For subsequent volumes in RAR 4.x, we can't easily extract the volume number
	// without fully parsing the header. Return a fallback value.
	// A better approach would be to look for ".partXX.rar" or ".rXX" in filename
	return -1 // Indicates we couldn't determine volume number reliably
}

// readVint reads a variable-length integer from RAR 5.0 format
// Returns the value and number of bytes read
func readVint(data []byte) (uint64, int) {
	if len(data) == 0 {
		return 0, 0
	}

	var value uint64
	var shift uint
	bytesRead := 0

	for i := 0; i < len(data) && i < 10; i++ { // Max 10 bytes for 64-bit int
		b := data[i]
		bytesRead++

		// Lower 7 bits contain data
		value |= uint64(b&0x7F) << shift
		shift += 7

		// Highest bit is continuation flag
		if b&0x80 == 0 {
			break // Last byte
		}
	}

	return value, bytesRead
}

// PostProcess handles post-download processing like file detection and extraction
// This is called separately after download completes to allow next download to start
func (fd *FastDownloader) PostProcess(downloadDir string) error {
	// Get list of downloaded files
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return fmt.Errorf("failed to read download directory: %v", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(downloadDir, entry.Name()))
		}
	}

	return fd.postProcess(files, downloadDir)
}

// postProcess handles post-download processing like file detection and extraction
func (fd *FastDownloader) postProcess(files []string, downloadDir string) error {
	if len(files) == 0 {
		return nil
	}

	fd.download.AddLog("Detecting file types and renaming...")

	// First pass: detect all RAR files and determine their volume numbers
	type rarFileInfo struct {
		path        string
		volumeIndex int // 0-based volume index (0 = first volume, 1 = second, etc.)
	}
	rarFileInfos := []rarFileInfo{}

	for fileIdx, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		// Read enough of the header to detect RAR and get volume info
		header := make([]byte, 512)
		n, _ := f.Read(header)
		f.Close()

		if n >= 7 && (string(header[:6]) == "Rar!\x1a\x07" || string(header[:4]) == "Rar!") {
			// This is a RAR file
			// Try multiple strategies to determine volume number (in order of reliability):

			// 1. Try to extract from filename (.partXX.rar, .rXX, .XX.rar patterns)
			volumeIndex := parseVolumeFromFilename(filepath.Base(file))

			// 2. If filename parsing failed, try header parsing
			if volumeIndex < 0 {
				volumeIndex = parseRARVolumeNumber(header, n)
			}

			// 3. If both failed, use file index as last resort (preserves NZB order)
			if volumeIndex < 0 {
				volumeIndex = fileIdx
				fd.download.AddLog(fmt.Sprintf("Detected RAR file: %s (using NZB order: %d)", filepath.Base(file), volumeIndex+1))
			} else {
				fd.download.AddLog(fmt.Sprintf("Detected RAR file: %s (volume %d)", filepath.Base(file), volumeIndex+1))
			}

			rarFileInfos = append(rarFileInfos, rarFileInfo{
				path:        file,
				volumeIndex: volumeIndex,
			})
		}
	}

	// Sort RAR files by volume index (parsed from header)
	sort.Slice(rarFileInfos, func(i, j int) bool {
		return rarFileInfos[i].volumeIndex < rarFileInfos[j].volumeIndex
	})

	// Extract just the paths in correct order
	rarFiles := []string{}
	for _, info := range rarFileInfos {
		rarFiles = append(rarFiles, info.path)
	}

	// Track if we have RAR archive for extraction
	var firstArchive string
	archiveType := ""

	// If we have multiple RAR files, rename them to proper volume naming for unrar
	if len(rarFiles) > 1 {
		fd.download.AddLog(fmt.Sprintf("Found %d RAR volumes, renaming for extraction...", len(rarFiles)))

		// Determine base name from download name or first file
		baseName := "archive"
		if fd.download != nil && fd.download.Name != "" {
			baseName = fd.download.Name
		}

		// Rename files to proper RAR volume naming (.part01.rar, .part02.rar, etc.)
		renamedRarFiles := []string{}
		renameFailed := false
		for i, file := range rarFiles {
			newName := filepath.Join(downloadDir, fmt.Sprintf("%s.part%02d.rar", baseName, i+1))
			if err := os.Rename(file, newName); err != nil {
				fd.download.AddLog(fmt.Sprintf("ERROR: Failed to rename RAR volume %d (%s -> %s): %v",
					i+1, filepath.Base(file), filepath.Base(newName), err))
				renameFailed = true
				break
			} else {
				if i < 5 || i >= len(rarFiles)-5 {
					// Log first 5 and last 5 to avoid spam
					fd.download.AddLog(fmt.Sprintf("Renamed volume %d: %s", i+1, filepath.Base(newName)))
				} else if i == 5 {
					fd.download.AddLog(fmt.Sprintf("... renaming volumes 6-%d ...", len(rarFiles)-5))
				}
				renamedRarFiles = append(renamedRarFiles, newName)
			}
		}

		if renameFailed {
			return fmt.Errorf("failed to rename all RAR volumes for extraction")
		}

		// Verify all files were renamed
		if len(renamedRarFiles) != len(rarFiles) {
			return fmt.Errorf("only renamed %d of %d RAR volumes", len(renamedRarFiles), len(rarFiles))
		}

		// Set first archive for extraction
		firstArchive = renamedRarFiles[0]
		archiveType = "rar"

		fd.download.AddLog(fmt.Sprintf("Successfully renamed all %d volumes, first volume: %s",
			len(renamedRarFiles), filepath.Base(firstArchive)))

		// Verify the first archive file exists before extraction
		if _, err := os.Stat(firstArchive); os.IsNotExist(err) {
			fd.download.AddLog(fmt.Sprintf("ERROR: First archive file missing after rename: %s", firstArchive))
			// List what files DO exist in the directory
			entries, _ := os.ReadDir(downloadDir)
			fd.download.AddLog(fmt.Sprintf("Files in download dir (%d):", len(entries)))
			for i, entry := range entries {
				if i < 10 || i >= len(entries)-10 {
					fd.download.AddLog(fmt.Sprintf("  %s", entry.Name()))
				} else if i == 10 {
					fd.download.AddLog(fmt.Sprintf("  ... %d more files ...", len(entries)-20))
				}
			}
			return fmt.Errorf("first archive file not found after rename: %s", firstArchive)
		}

		fd.download.AddLog(fmt.Sprintf("Verified first volume exists: %s", firstArchive))

		// Skip the file type detection for multi-volume RARs - we'll extract directly
		fd.download.AddLog(fmt.Sprintf("Detected %s archive, extracting...", archiveType))

		// Try extracting with common passwords
		output, err := fd.extractRARWithPassword(firstArchive, downloadDir)
		if err != nil {
			fd.download.AddLog(fmt.Sprintf("Extraction failed: %v", err))

			// Log full output for debugging (split into chunks if needed)
			outputStr := string(output)
			if len(outputStr) > 0 {
				// Log in chunks of 500 characters
				for i := 0; i < len(outputStr); i += 500 {
					end := i + 500
					if end > len(outputStr) {
						end = len(outputStr)
					}
					fd.download.AddLog(fmt.Sprintf("unrar output [%d-%d]: %s", i, end, outputStr[i:end]))
				}
			}

			if strings.Contains(outputStr, "previous volume") || strings.Contains(outputStr, "Unexpected end") {
				fd.download.AddLog("Archive appears incomplete - missing volumes or damaged files")
				return fmt.Errorf("archive extraction failed: incomplete archive - missing volumes or damaged files")
			} else if strings.Contains(outputStr, "CRC failed") {
				fd.download.AddLog("Archive is corrupted - CRC check failed")
				return fmt.Errorf("archive extraction failed: CRC check failed - corrupted archive")
			} else if strings.Contains(outputStr, "cannot find volume") {
				fd.download.AddLog("Missing archive volumes - multipart archive is incomplete")
				return fmt.Errorf("archive extraction failed: missing archive volumes")
			}

			// Generic extraction failure
			return fmt.Errorf("archive extraction failed: %v", err)
		}

		fd.download.AddLog("Extraction complete")

		// Clean up archive files and auxiliary files after successful extraction
		fd.download.AddLog("Cleaning up archive and auxiliary files...")
		for _, file := range renamedRarFiles {
			os.Remove(file)
		}

		// Clean up auxiliary files
		fd.cleanupAuxiliaryFiles(downloadDir)

		return nil
	} else if len(rarFiles) == 1 {
		// Single RAR file, process normally
		firstArchive = rarFiles[0]
		archiveType = "rar"
	}

	// Detect file types and rename (for non-RAR files or single RAR)
	renamedFiles := []string{}

	for _, file := range files {
		// Read file header to detect type
		f, err := os.Open(file)
		if err != nil {
			renamedFiles = append(renamedFiles, file)
			continue
		}
		header := make([]byte, 16)
		n, _ := f.Read(header)
		f.Close()

		if n < 4 {
			renamedFiles = append(renamedFiles, file)
			continue
		}

		var newName string
		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		if strings.Contains(baseName, "-part") {
			baseName = baseName[:strings.Index(baseName, "-part")]
		}

		// Detect file type by magic bytes
		if n >= 4 && header[0] == 0x1A && header[1] == 0x45 && header[2] == 0xDF && header[3] == 0xA3 {
			// Matroska/WebM (MKV)
			newName = filepath.Join(downloadDir, baseName+".mkv")
			fd.download.AddLog(fmt.Sprintf("Detected video file: %s", filepath.Base(newName)))
		} else if n >= 8 && string(header[4:8]) == "ftyp" {
			// MP4/M4V
			newName = filepath.Join(downloadDir, baseName+".mp4")
			fd.download.AddLog(fmt.Sprintf("Detected video file: %s", filepath.Base(newName)))
		} else if n >= 4 && string(header[:4]) == "RIFF" {
			// AVI
			newName = filepath.Join(downloadDir, baseName+".avi")
			fd.download.AddLog(fmt.Sprintf("Detected video file: %s", filepath.Base(newName)))
		} else if n >= 7 && (string(header[:6]) == "Rar!\x1a\x07" || string(header[:4]) == "Rar!") {
			// RAR archive (already concatenated if multi-part)
			newName = file
			if firstArchive == "" {
				firstArchive = newName
				archiveType = "rar"
			}
		} else if n >= 4 && string(header[:4]) == "PK\x03\x04" {
			// ZIP archive
			newName = filepath.Join(downloadDir, baseName+".zip")
			if firstArchive == "" {
				firstArchive = newName
				archiveType = "zip"
			}
		} else if n >= 6 && string(header[:6]) == "7z\xbc\xaf\x27\x1c" {
			// 7z archive
			newName = filepath.Join(downloadDir, baseName+".7z")
			if firstArchive == "" {
				firstArchive = newName
				archiveType = "7z"
			}
		} else {
			// Unknown type, keep as-is
			newName = file
		}

		// Rename file if needed
		if newName != file && newName != "" {
			if err := os.Rename(file, newName); err != nil {
				fd.download.AddLog(fmt.Sprintf("Failed to rename %s: %v", filepath.Base(file), err))
				renamedFiles = append(renamedFiles, file)
			} else {
				fd.download.AddLog(fmt.Sprintf("Renamed: %s -> %s", filepath.Base(file), filepath.Base(newName)))
				renamedFiles = append(renamedFiles, newName)
			}
		} else {
			renamedFiles = append(renamedFiles, file)
		}
	}

	if firstArchive == "" {
		fd.download.AddLog("No archives detected, files ready")
		// Still clean up auxiliary files even without extraction
		fd.cleanupAuxiliaryFiles(downloadDir)
		return nil
	}

	fd.download.AddLog(fmt.Sprintf("Detected %s archive, extracting...", archiveType))

	// Extract based on type
	var output []byte
	var err error
	switch archiveType {
	case "rar":
		output, err = fd.extractRARWithPassword(firstArchive, downloadDir)
	case "zip":
		cmd := exec.Command("unzip", "-o", firstArchive, "-d", downloadDir)
		output, err = cmd.CombinedOutput()
	case "7z":
		cmd := exec.Command("7z", "x", "-o"+downloadDir, "-y", firstArchive)
		output, err = cmd.CombinedOutput()
	default:
		return nil
	}
	if err != nil {
		fd.download.AddLog(fmt.Sprintf("Extraction failed: %v", err))

		// Log full output for debugging (split into chunks if needed)
		outputStr := string(output)
		if len(outputStr) > 0 {
			// Log in chunks of 500 characters
			for i := 0; i < len(outputStr); i += 500 {
				end := i + 500
				if end > len(outputStr) {
					end = len(outputStr)
				}
				fd.download.AddLog(fmt.Sprintf("unrar output [%d-%d]: %s", i, end, outputStr[i:end]))
			}
		}

		if strings.Contains(outputStr, "previous volume") || strings.Contains(outputStr, "Unexpected end") {
			fd.download.AddLog("Archive appears incomplete - missing volumes or damaged files")
			return fmt.Errorf("archive extraction failed: incomplete archive - missing volumes or damaged files")
		} else if strings.Contains(outputStr, "CRC failed") {
			fd.download.AddLog("Archive is corrupted - CRC check failed")
			return fmt.Errorf("archive extraction failed: CRC check failed - corrupted archive")
		} else if strings.Contains(outputStr, "cannot find volume") {
			fd.download.AddLog("Missing archive volumes - multipart archive is incomplete")
			return fmt.Errorf("archive extraction failed: missing archive volumes")
		}

		// Generic extraction failure
		return fmt.Errorf("archive extraction failed: %v", err)
	}

	fd.download.AddLog("Extraction complete")

	// Clean up archive files and auxiliary files after successful extraction
	fd.download.AddLog("Cleaning up archive and auxiliary files...")
	for _, file := range renamedFiles {
		os.Remove(file)
	}

	// Clean up auxiliary files
	fd.cleanupAuxiliaryFiles(downloadDir)

	return nil
}

// extractRARWithPassword attempts to extract a RAR file, trying common passwords
func (fd *FastDownloader) extractRARWithPassword(rarFile, destDir string) ([]byte, error) {
	// Build list of passwords to try
	passwords := []string{}

	// First priority: password from NZB file metadata
	if fd.download.NZBData != nil && fd.download.NZBData.Password != "" {
		passwords = append(passwords, fd.download.NZBData.Password)
		fd.download.AddLog(fmt.Sprintf("Found password in NZB metadata: %s", fd.download.NZBData.Password))
	}

	// Second priority: try without password
	passwords = append(passwords, "")

	// Third priority: common scene/indexer passwords
	if fd.download.Metadata != nil {
		// Try indexer name
		if indexerName, ok := fd.download.Metadata["indexer_name"].(string); ok && indexerName != "" {
			passwords = append(passwords, indexerName)
			passwords = append(passwords, strings.ToLower(indexerName))
		}

		// Try indexer ID
		if indexerID, ok := fd.download.Metadata["indexer_id"].(string); ok && indexerID != "" {
			passwords = append(passwords, indexerID)
			passwords = append(passwords, strings.ToLower(indexerID))
		}
	}

	// Extract release group from download name (usually at the end after a dash)
	if fd.download.Name != "" {
		parts := strings.Split(fd.download.Name, "-")
		if len(parts) > 1 {
			releaseGroup := strings.TrimSpace(parts[len(parts)-1])
			passwords = append(passwords, releaseGroup)
			passwords = append(passwords, strings.ToLower(releaseGroup))
		}
	}

	// Common generic passwords
	commonPasswords := []string{
		"nzbgeek", "dognzb", "nzbsu", "nzbfinder",
		"password", "usenet", "scene",
	}
	passwords = append(passwords, commonPasswords...)

	// Try each password
	var lastOutput []byte
	var lastErr error

	for i, password := range passwords {
		if i == 0 {
			fd.download.AddLog("Attempting extraction without password...")
		} else {
			fd.download.AddLog(fmt.Sprintf("Trying password: %s", password))
		}

		// Build unrar command with password
		args := []string{"x", "-o+", "-y"}
		if password != "" {
			args = append(args, fmt.Sprintf("-p%s", password))
		} else {
			args = append(args, "-p-") // -p- means no password
		}
		args = append(args, rarFile, destDir+"/")

		cmd := exec.Command("unrar", args...)
		output, err := cmd.CombinedOutput()

		lastOutput = output
		lastErr = err

		if err == nil {
			// Success!
			if password != "" {
				fd.download.AddLog(fmt.Sprintf("Successfully extracted with password: %s", password))
			} else {
				fd.download.AddLog("Successfully extracted (no password)")
			}
			return output, nil
		}

		// Check if error is password-related
		outputStr := string(output)
		if !strings.Contains(outputStr, "password") &&
			!strings.Contains(outputStr, "encrypted") &&
			!strings.Contains(outputStr, "Enter password") {
			// Error is not password-related, stop trying
			fd.download.AddLog("Extraction error is not password-related, stopping password attempts")
			break
		}
	}

	// All passwords failed
	fd.download.AddLog(fmt.Sprintf("Failed to extract after trying %d password(s)", len(passwords)))
	return lastOutput, lastErr
}

// cleanupAuxiliaryFiles removes common auxiliary files (.nfo, .sfv, .bin, samples, etc.)
func (fd *FastDownloader) cleanupAuxiliaryFiles(downloadDir string) {
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		ext := strings.ToLower(filepath.Ext(filename))

		// Remove auxiliary files
		shouldRemove := false
		if ext == ".nfo" || ext == ".sfv" || ext == ".nzb" || ext == ".txt" {
			shouldRemove = true
		} else if strings.Contains(strings.ToLower(filename), "sample") {
			shouldRemove = true
		} else if strings.HasSuffix(filename, ".bin") {
			// Remove .bin files that were likely temporary parts
			shouldRemove = true
		}

		if shouldRemove {
			fullPath := filepath.Join(downloadDir, filename)
			if err := os.Remove(fullPath); err == nil {
				fd.download.AddLog(fmt.Sprintf("Removed auxiliary file: %s", filename))
			}
		}
	}
}

// Close closes the downloader and all connections
func (fd *FastDownloader) Close() {
	// Cancel context first to signal workers to stop
	fd.cancel()

	// Close job queue to prevent new jobs
	close(fd.jobQueue)

	// Wait for workers to finish with a timeout
	done := make(chan struct{})
	go func() {
		fd.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Workers finished normally
	case <-time.After(5 * time.Second):
		// Workers didn't finish in time, force close
		fd.download.AddLog("WARNING: Forcing connection closure after timeout")
	}

	// Close all connections regardless
	for i, conn := range fd.connPool {
		if conn != nil {
			if err := conn.Close(); err != nil {
				fd.download.AddLog(fmt.Sprintf("Error closing connection %d: %v", i, err))
			}
		}
	}

	// Clear the connection pool
	fd.connPool = nil
}

// FileAssembler handles writing segments to a file in order
type FileAssembler struct {
	file          *os.File
	filepath      string
	segments      []bool
	totalSegments int
	mu            sync.Mutex
	buffer        map[int][]byte
}

// NewFileAssembler creates a new file assembler
func NewFileAssembler(path string, totalSegments int) (*FileAssembler, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return &FileAssembler{
		file:          file,
		filepath:      path,
		segments:      make([]bool, totalSegments),
		totalSegments: totalSegments,
		buffer:        make(map[int][]byte),
	}, nil
}

// WriteSegment writes a segment to the file
func (fa *FileAssembler) WriteSegment(index int, data []byte) error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	if index < 0 || index >= fa.totalSegments {
		return fmt.Errorf("invalid segment index: %d", index)
	}

	if fa.segments[index] {
		return nil // Already written
	}

	// Buffer the segment
	fa.buffer[index] = data
	fa.segments[index] = true

	// Write any sequential segments from the buffer
	return fa.flushSequential()
}

// flushSequential writes all sequential segments that are ready
func (fa *FileAssembler) flushSequential() error {
	// Find the first index that hasn't been written to disk yet
	writeIndex := 0
	for writeIndex < fa.totalSegments && fa.segments[writeIndex] {
		writeIndex++
	}

	// Actually write to disk starting from 0
	for i := 0; i < writeIndex; i++ {
		if data, ok := fa.buffer[i]; ok {
			if _, err := fa.file.Write(data); err != nil {
				return err
			}
			delete(fa.buffer, i)
		}
	}

	return nil
}

// Close finalizes the file
func (fa *FileAssembler) Close() error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	// Write any remaining buffered segments in order
	for i := 0; i < fa.totalSegments; i++ {
		if data, ok := fa.buffer[i]; ok {
			if _, err := fa.file.Write(data); err != nil {
				return err
			}
		}
	}

	return fa.file.Close()
}
