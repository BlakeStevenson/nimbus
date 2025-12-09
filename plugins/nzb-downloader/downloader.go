package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
		return nil, fmt.Errorf("failed to establish any NNTP connections")
	}

	download.AddLog(fmt.Sprintf("Created %d/%d connections successfully", len(fd.connPool), numConnections))

	// Start workers
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

	// Sort files by name to ensure proper order
	sort.Strings(files)

	// First pass: detect all RAR files and group them
	type rarGroup struct {
		baseName string
		parts    map[int]string // part number -> filename
	}
	rarGroups := make(map[string]*rarGroup) // baseName -> group

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		header := make([]byte, 16)
		n, _ := f.Read(header)
		f.Close()

		if n >= 7 && (string(header[:6]) == "Rar!\x1a\x07" || string(header[:4]) == "Rar!") {
			// This is a RAR file, extract base name and part number
			baseName := filepath.Base(file)
			partNum := 1
			baseNameWithoutPart := baseName

			if idx := strings.Index(baseName, "-part"); idx != -1 {
				baseNameWithoutPart = baseName[:idx]
				fmt.Sscanf(baseName[idx+5:], "%d", &partNum)
			}

			if rarGroups[baseNameWithoutPart] == nil {
				rarGroups[baseNameWithoutPart] = &rarGroup{
					baseName: baseNameWithoutPart,
					parts:    make(map[int]string),
				}
			}
			rarGroups[baseNameWithoutPart].parts[partNum] = file
		}
	}

	// Concatenate and rename RAR volumes if we have multiple parts
	for baseName, group := range rarGroups {
		if len(group.parts) > 1 {
			fd.download.AddLog(fmt.Sprintf("Found %d RAR parts for %s, concatenating into volumes...", len(group.parts), baseName))

			// Sort part numbers
			partNums := make([]int, 0, len(group.parts))
			for partNum := range group.parts {
				partNums = append(partNums, partNum)
			}
			sort.Ints(partNums)

			// Concatenate all parts into a single RAR file
			rarPath := filepath.Join(downloadDir, baseName+".rar")
			outFile, err := os.Create(rarPath)
			if err != nil {
				fd.download.AddLog(fmt.Sprintf("ERROR: Failed to create concatenated RAR: %v", err))
				continue
			}

			for _, partNum := range partNums {
				partFile := group.parts[partNum]
				fd.download.AddLog(fmt.Sprintf("Concatenating part%d...", partNum))

				inFile, err := os.Open(partFile)
				if err != nil {
					fd.download.AddLog(fmt.Sprintf("ERROR: Failed to open part %d: %v", partNum, err))
					outFile.Close()
					os.Remove(rarPath)
					continue
				}

				if _, err := io.Copy(outFile, inFile); err != nil {
					fd.download.AddLog(fmt.Sprintf("ERROR: Failed to concatenate part %d: %v", partNum, err))
					inFile.Close()
					outFile.Close()
					os.Remove(rarPath)
					continue
				}

				inFile.Close()
				os.Remove(partFile) // Remove the part file after concatenation
			}

			outFile.Close()
			fd.download.AddLog(fmt.Sprintf("Created concatenated RAR: %s", filepath.Base(rarPath)))

			// Update files list - remove all parts and add the concatenated file
			newFiles := []string{}
			for _, f := range files {
				isPartFile := false
				for _, partFile := range group.parts {
					if f == partFile {
						isPartFile = true
						break
					}
				}
				if !isPartFile {
					newFiles = append(newFiles, f)
				}
			}
			newFiles = append(newFiles, rarPath)
			files = newFiles
		}
	}

	// Detect file types and rename
	renamedFiles := []string{}
	var firstArchive string
	archiveType := ""

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
	var cmd *exec.Cmd
	switch archiveType {
	case "rar":
		cmd = exec.Command("unrar", "x", "-o+", "-y", firstArchive, downloadDir+"/")
	case "zip":
		cmd = exec.Command("unzip", "-o", firstArchive, "-d", downloadDir)
	case "7z":
		cmd = exec.Command("7z", "x", "-o"+downloadDir, "-y", firstArchive)
	default:
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fd.download.AddLog(fmt.Sprintf("Extraction failed: %v", err))
		if strings.Contains(string(output), "previous volume") || strings.Contains(string(output), "Unexpected end") {
			fd.download.AddLog("Archive appears incomplete - missing volumes or damaged files")
			fd.download.AddLog("Check if PAR2 repair files are available, or manually extract")
		} else {
			fd.download.AddLog(fmt.Sprintf("Error details: %s", string(output)[:min(200, len(output))]))
		}
		// Don't clean up files if extraction failed
		return nil // Return nil so download still shows as completed
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
	fd.cancel()
	close(fd.jobQueue)

	// Wait for workers to finish
	fd.wg.Wait()

	// Close all connections
	for _, conn := range fd.connPool {
		conn.Close()
	}
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
