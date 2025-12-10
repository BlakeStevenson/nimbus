import { useState, useEffect, useMemo } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";

import { Checkbox } from "@/components/ui/checkbox";
import {
  Download,
  Loader2,
  Search,
  AlertCircle,
  X,
  ExternalLink,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  TrendingUp,
  Check,
  Ban,
} from "lucide-react";
import { useInteractiveSearch, type IndexerRelease } from "@/lib/api/media";
import { useCreateDownload, useDownloaders } from "@/lib/api/downloads";
import { useToast } from "@/hooks/use-toast";
import {
  useMediaQuality,
  useDetectQuality,
  useCheckUpgrade,
} from "@/lib/api/quality";
import { formatDistanceToNow } from "date-fns";

interface InteractiveSearchDialogProps {
  mediaId: string | number;
  mediaTitle: string;
  mediaKind: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelectRelease?: (release: IndexerRelease) => void;
}

type SortColumn = "title" | "size" | "age" | "indexer";
type SortDirection = "asc" | "desc" | null;

interface MultiSelectProps {
  label: string;
  options: string[];
  selected: Set<string>;
  onChange: (selected: Set<string>) => void;
}

function MultiSelect({ label, options, selected, onChange }: MultiSelectProps) {
  const [open, setOpen] = useState(false);

  const toggleOption = (option: string) => {
    const newSelected = new Set(selected);
    if (newSelected.has(option)) {
      newSelected.delete(option);
    } else {
      newSelected.add(option);
    }
    onChange(newSelected);
  };

  const displayText =
    selected.size === 0
      ? label
      : selected.size === options.length
        ? `${label}: All`
        : `${label}: ${selected.size}`;

  return (
    <div className="relative">
      <Button
        variant="outline"
        size="sm"
        className="h-9 text-xs"
        onClick={() => setOpen(!open)}
      >
        {displayText}
      </Button>
      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute top-full left-0 z-50 mt-1 w-48 rounded-md border bg-popover p-2 shadow-md">
            <div className="space-y-2">
              {options.map((option) => (
                <div key={option} className="flex items-center space-x-2">
                  <Checkbox
                    id={`${label}-${option}`}
                    checked={selected.has(option)}
                    onCheckedChange={() => toggleOption(option)}
                  />
                  <label
                    htmlFor={`${label}-${option}`}
                    className="text-sm cursor-pointer"
                  >
                    {option}
                  </label>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export function InteractiveSearchDialog({
  mediaId,
  mediaTitle,
  mediaKind,
  open,
  onOpenChange,
  onSelectRelease,
}: InteractiveSearchDialogProps) {
  const [searchFilter, setSearchFilter] = useState("");
  const [downloadingReleases, setDownloadingReleases] = useState<Set<string>>(
    new Set(),
  );
  const [sortColumn, setSortColumn] = useState<SortColumn | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>(null);
  const [selectedQualities, setSelectedQualities] = useState<Set<string>>(
    new Set(),
  );
  const [selectedVideoCodecs, setSelectedVideoCodecs] = useState<Set<string>>(
    new Set(),
  );
  const [selectedAudioCodecs, setSelectedAudioCodecs] = useState<Set<string>>(
    new Set(),
  );

  const { toast } = useToast();

  const {
    data: searchResults,
    isLoading,
    error,
    refetch,
  } = useInteractiveSearch(mediaId);

  const { data: downloadersData } = useDownloaders();
  const createDownload = useCreateDownload();

  // Quality profile integration
  const { data: mediaQuality } = useMediaQuality(Number(mediaId));
  const detectQuality = useDetectQuality();

  // Trigger search when dialog opens
  useEffect(() => {
    if (open) {
      refetch();
    }
  }, [open, refetch]);

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const units = ["B", "KB", "MB", "GB", "TB"];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  const getQualityBadge = (title: string) => {
    const lower = title.toLowerCase();
    if (lower.includes("2160p") || lower.includes("4k")) return "4K";
    if (lower.includes("1080p")) return "1080p";
    if (lower.includes("720p")) return "720p";
    if (lower.includes("480p")) return "480p";
    return "Unknown";
  };

  const getVideoCodec = (title: string): string => {
    const lower = title.toLowerCase();
    if (
      lower.includes("x265") ||
      lower.includes("hevc") ||
      lower.includes("h265")
    )
      return "x265";
    if (
      lower.includes("x264") ||
      lower.includes("avc") ||
      lower.includes("h264")
    )
      return "x264";
    if (lower.includes("xvid")) return "XviD";
    if (lower.includes("av1")) return "AV1";
    return "Unknown";
  };

  const getAudioCodec = (title: string): string => {
    const lower = title.toLowerCase();
    if (lower.includes("atmos")) return "Atmos";
    if (lower.includes("truehd")) return "TrueHD";
    if (lower.includes("dts-hd") || lower.includes("dts.hd")) return "DTS-HD";
    if (lower.includes("dts")) return "DTS";
    if (
      lower.includes("dd5.1") ||
      lower.includes("dd+5.1") ||
      lower.includes("ac3") ||
      lower.includes("ddp5.1")
    )
      return "DD5.1";
    if (lower.includes("aac")) return "AAC";
    if (lower.includes("mp3")) return "MP3";
    if (lower.includes("flac")) return "FLAC";
    return "Unknown";
  };

  const getCodecInfo = (title: string) => {
    const codecs = [];
    const videoCodec = getVideoCodec(title);
    const audioCodec = getAudioCodec(title);

    if (videoCodec !== "Unknown") codecs.push(videoCodec);
    if (audioCodec !== "Unknown") codecs.push(audioCodec);

    return codecs;
  };

  // Extract unique values for filters
  const filterOptions = useMemo(() => {
    if (!searchResults?.releases) {
      return {
        qualities: [] as string[],
        videoCodecs: [] as string[],
        audioCodecs: [] as string[],
      };
    }

    const qualities = new Set<string>();
    const videoCodecs = new Set<string>();
    const audioCodecs = new Set<string>();

    searchResults.releases.forEach((release) => {
      qualities.add(getQualityBadge(release.title));
      videoCodecs.add(getVideoCodec(release.title));
      audioCodecs.add(getAudioCodec(release.title));
    });

    return {
      qualities: Array.from(qualities).sort(),
      videoCodecs: Array.from(videoCodecs).sort(),
      audioCodecs: Array.from(audioCodecs).sort(),
    };
  }, [searchResults]);

  const handleSort = (column: SortColumn) => {
    if (sortColumn === column) {
      // Cycle through: asc -> desc -> null
      if (sortDirection === "asc") {
        setSortDirection("desc");
      } else if (sortDirection === "desc") {
        setSortDirection(null);
        setSortColumn(null);
      }
    } else {
      setSortColumn(column);
      setSortDirection("asc");
    }
  };

  const getSortIcon = (column: SortColumn) => {
    if (sortColumn !== column) {
      return <ArrowUpDown className="h-4 w-4 ml-1 inline" />;
    }
    if (sortDirection === "asc") {
      return <ArrowUp className="h-4 w-4 ml-1 inline" />;
    }
    if (sortDirection === "desc") {
      return <ArrowDown className="h-4 w-4 ml-1 inline" />;
    }
    return <ArrowUpDown className="h-4 w-4 ml-1 inline" />;
  };

  const filteredAndSortedReleases = useMemo(() => {
    let releases = searchResults?.releases || [];

    // Apply text filter
    releases = releases.filter((release) =>
      release.title.toLowerCase().includes(searchFilter.toLowerCase()),
    );

    // Apply quality filter
    if (selectedQualities.size > 0) {
      releases = releases.filter((release) =>
        selectedQualities.has(getQualityBadge(release.title)),
      );
    }

    // Apply video codec filter
    if (selectedVideoCodecs.size > 0) {
      releases = releases.filter((release) =>
        selectedVideoCodecs.has(getVideoCodec(release.title)),
      );
    }

    // Apply audio codec filter
    if (selectedAudioCodecs.size > 0) {
      releases = releases.filter((release) =>
        selectedAudioCodecs.has(getAudioCodec(release.title)),
      );
    }

    // Apply sorting
    if (sortColumn && sortDirection) {
      releases = [...releases].sort((a, b) => {
        let compareValue = 0;

        switch (sortColumn) {
          case "title":
            compareValue = a.title.localeCompare(b.title);
            break;
          case "size":
            compareValue = a.size - b.size;
            break;
          case "age":
            compareValue =
              new Date(a.publish_date).getTime() -
              new Date(b.publish_date).getTime();
            break;
          case "indexer":
            compareValue = a.indexer_name.localeCompare(b.indexer_name);
            break;
        }

        return sortDirection === "asc" ? compareValue : -compareValue;
      });
    }

    return releases;
  }, [
    searchResults,
    searchFilter,
    selectedQualities,
    selectedVideoCodecs,
    selectedAudioCodecs,
    sortColumn,
    sortDirection,
  ]);

  const handleDownload = async (release: IndexerRelease) => {
    // Find an NZB downloader
    const nzbDownloader = downloadersData?.downloaders.find(
      (d) => d.id === "nzb-downloader",
    );

    if (!nzbDownloader) {
      toast({
        title: "No downloader available",
        description: "NZB downloader plugin is not available",
        variant: "error",
      });
      return;
    }

    // Mark as downloading
    setDownloadingReleases((prev) => new Set(prev).add(release.guid));

    try {
      await createDownload.mutateAsync({
        plugin_id: nzbDownloader.id,
        name: release.title,
        url: release.download_url,
        priority: 0,
        metadata: {
          indexer_id: release.indexer_id,
          indexer_name: release.indexer_name,
          size: release.size,
          media_id: mediaId,
          media_title: mediaTitle,
          media_kind: mediaKind,
        },
      });

      toast({
        title: "Download started",
        description: `${release.title} has been added to the download queue`,
        variant: "success",
      });

      if (onSelectRelease) {
        onSelectRelease(release);
      }

      // Close the modal after successful download
      onOpenChange(false);
    } catch (err) {
      toast({
        title: "Download failed",
        description:
          err instanceof Error ? err.message : "Failed to start download",
        variant: "error",
      });
    } finally {
      // Remove from downloading set after a delay
      setTimeout(() => {
        setDownloadingReleases((prev) => {
          const next = new Set(prev);
          next.delete(release.guid);
          return next;
        });
      }, 2000);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Search className="h-5 w-5" />
            Interactive Search: {mediaTitle}
          </DialogTitle>
          <DialogDescription>
            Search results from {searchResults?.sources.length || 0} indexer(s)
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <Input
              placeholder="Filter releases..."
              value={searchFilter}
              onChange={(e) => setSearchFilter(e.target.value)}
              className="flex-1"
            />
            <Badge variant="outline">
              {filteredAndSortedReleases.length} of {searchResults?.total || 0}{" "}
              releases
            </Badge>
          </div>

          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm text-muted-foreground">Filters:</span>
            <MultiSelect
              label="Quality"
              options={filterOptions.qualities}
              selected={selectedQualities}
              onChange={setSelectedQualities}
            />
            <MultiSelect
              label="Video Codec"
              options={filterOptions.videoCodecs}
              selected={selectedVideoCodecs}
              onChange={setSelectedVideoCodecs}
            />
            <MultiSelect
              label="Audio Codec"
              options={filterOptions.audioCodecs}
              selected={selectedAudioCodecs}
              onChange={setSelectedAudioCodecs}
            />
            {(selectedQualities.size > 0 ||
              selectedVideoCodecs.size > 0 ||
              selectedAudioCodecs.size > 0) && (
              <Button
                variant="ghost"
                size="sm"
                className="h-9 text-xs"
                onClick={() => {
                  setSelectedQualities(new Set());
                  setSelectedVideoCodecs(new Set());
                  setSelectedAudioCodecs(new Set());
                }}
              >
                Clear filters
              </Button>
            )}
          </div>
        </div>

        <div className="flex-1 overflow-auto">
          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">
                Searching indexers...
              </span>
            </div>
          )}

          {error && (
            <div className="flex items-center justify-center py-12 text-destructive">
              <AlertCircle className="h-8 w-8 mr-2" />
              <span>Failed to search: {(error as Error).message}</span>
            </div>
          )}

          {!isLoading && !error && filteredAndSortedReleases.length === 0 && (
            <div className="flex items-center justify-center py-12 text-muted-foreground">
              <AlertCircle className="h-8 w-8 mr-2" />
              <span>No releases found</span>
            </div>
          )}

          {!isLoading && !error && filteredAndSortedReleases.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead
                    className="cursor-pointer select-none"
                    onClick={() => handleSort("title")}
                  >
                    Release{getSortIcon("title")}
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none"
                    onClick={() => handleSort("size")}
                  >
                    Size{getSortIcon("size")}
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none"
                    onClick={() => handleSort("age")}
                  >
                    Age{getSortIcon("age")}
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none"
                    onClick={() => handleSort("indexer")}
                  >
                    Indexer{getSortIcon("indexer")}
                  </TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredAndSortedReleases.map((release) => {
                  const quality = getQualityBadge(release.title);
                  const codecs = getCodecInfo(release.title);
                  const publishDate = new Date(release.publish_date);
                  const age = formatDistanceToNow(publishDate, {
                    addSuffix: true,
                  });

                  return (
                    <TableRow key={release.guid}>
                      <TableCell>
                        <div className="space-y-1">
                          <div className="font-medium text-sm">
                            {release.title}
                          </div>
                          <div className="flex items-center gap-1 flex-wrap">
                            {quality && (
                              <Badge variant="secondary" className="text-xs">
                                {quality}
                              </Badge>
                            )}
                            {codecs.map((codec) => (
                              <Badge
                                key={codec}
                                variant="outline"
                                className="text-xs"
                              >
                                {codec}
                              </Badge>
                            ))}
                            {release.attributes?.season && (
                              <Badge variant="outline" className="text-xs">
                                S{release.attributes.season.padStart(2, "0")}
                                {release.attributes.episode &&
                                  `E${release.attributes.episode.padStart(2, "0")}`}
                              </Badge>
                            )}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="text-sm">
                        {formatFileSize(release.size)}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {age}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">
                          {release.indexer_name}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          {release.link && (
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() =>
                                window.open(release.link, "_blank")
                              }
                            >
                              <ExternalLink className="h-4 w-4" />
                            </Button>
                          )}
                          <Button
                            size="sm"
                            onClick={() => handleDownload(release)}
                            disabled={downloadingReleases.has(release.guid)}
                          >
                            {downloadingReleases.has(release.guid) ? (
                              <>
                                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                                Adding...
                              </>
                            ) : (
                              <>
                                <Download className="h-4 w-4 mr-1" />
                                Download
                              </>
                            )}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </div>

        <div className="flex justify-end gap-2 pt-4 border-t">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            <X className="h-4 w-4 mr-1" />
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
