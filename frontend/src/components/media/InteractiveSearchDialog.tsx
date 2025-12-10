import { useState, useEffect } from "react";
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
import {
  Download,
  Loader2,
  Search,
  AlertCircle,
  CheckCircle,
  X,
  ExternalLink,
} from "lucide-react";
import { useInteractiveSearch, type IndexerRelease } from "@/lib/api/media";
import { formatDistanceToNow } from "date-fns";

interface InteractiveSearchDialogProps {
  mediaId: string | number;
  mediaTitle: string;
  mediaKind: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelectRelease?: (release: IndexerRelease) => void;
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
  const [selectedRelease, setSelectedRelease] = useState<string | null>(null);

  const {
    data: searchResults,
    isLoading,
    error,
    refetch,
  } = useInteractiveSearch(mediaId);

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
    return null;
  };

  const getCodecInfo = (title: string) => {
    const lower = title.toLowerCase();
    const codecs = [];
    if (lower.includes("x265") || lower.includes("hevc")) codecs.push("x265");
    else if (lower.includes("x264") || lower.includes("avc")) codecs.push("x264");
    if (lower.includes("dts")) codecs.push("DTS");
    else if (lower.includes("ac3") || lower.includes("dd5.1")) codecs.push("DD5.1");
    return codecs;
  };

  const filteredReleases =
    searchResults?.releases.filter((release) =>
      release.title.toLowerCase().includes(searchFilter.toLowerCase())
    ) || [];

  const handleDownload = (release: IndexerRelease) => {
    setSelectedRelease(release.guid);
    if (onSelectRelease) {
      onSelectRelease(release);
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

        <div className="flex items-center gap-2 mb-4">
          <Input
            placeholder="Filter releases..."
            value={searchFilter}
            onChange={(e) => setSearchFilter(e.target.value)}
            className="flex-1"
          />
          <Badge variant="outline">
            {filteredReleases.length} of {searchResults?.total || 0} releases
          </Badge>
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

          {!isLoading && !error && filteredReleases.length === 0 && (
            <div className="flex items-center justify-center py-12 text-muted-foreground">
              <AlertCircle className="h-8 w-8 mr-2" />
              <span>No releases found</span>
            </div>
          )}

          {!isLoading && !error && filteredReleases.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Release</TableHead>
                  <TableHead>Size</TableHead>
                  <TableHead>Age</TableHead>
                  <TableHead>Indexer</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredReleases.map((release) => {
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
                              onClick={() => window.open(release.link, "_blank")}
                            >
                              <ExternalLink className="h-4 w-4" />
                            </Button>
                          )}
                          <Button
                            size="sm"
                            onClick={() => handleDownload(release)}
                            disabled={selectedRelease === release.guid}
                          >
                            {selectedRelease === release.guid ? (
                              <CheckCircle className="h-4 w-4 mr-1" />
                            ) : (
                              <Download className="h-4 w-4 mr-1" />
                            )}
                            {selectedRelease === release.guid
                              ? "Selected"
                              : "Download"}
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
