import { useState, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  useMediaItem,
  useUpdateMedia,
  useMediaFiles,
  useDeleteMediaFile,
  useDeleteMediaItem,
  useMediaList,
  type IndexerRelease,
} from "@/lib/api/media";
import { useTVSeasonDetails } from "@/lib/api/tmdb";
import { useDownloads } from "@/lib/api/downloads";
import {
  useQualityProfiles,
  useMediaQuality,
  useAssignProfileToMedia,
  useQualityUpgradeHistory,
} from "@/lib/api/quality";
import {
  useMonitoringRuleByMedia,
  useCreateMonitoringRule,
  useUpdateMonitoringRule,
  useDeleteMonitoringRule,
} from "@/lib/api/monitoring";
import { MediaKindBadge } from "@/components/media/MediaKindBadge";
import { MediaGrid } from "@/components/media/MediaGrid";
import { MediaTable } from "@/components/media/MediaTable";
import { InteractiveSearchDialog } from "@/components/media/InteractiveSearchDialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  ArrowLeft,
  Loader2,
  Pencil,
  Trash2,
  X,
  File,
  HardDrive,
  LayoutGrid,
  Table as TableIcon,
  Star,
  Clock,
  Calendar,
  Plus,
  CheckCircle,
  XCircle,
  Download,
  Search,
  Award,
  TrendingUp,
  AlertCircle,
  Activity,
  Eye,
  EyeOff,
} from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDate } from "@/lib/utils";

export function MediaDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [deleteConfirmed, setDeleteConfirmed] = useState(false);
  const [isFileDeleteOpen, setIsFileDeleteOpen] = useState(false);
  const [fileToDelete, setFileToDelete] = useState<number | null>(null);
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [searchEpisodeId, setSearchEpisodeId] = useState<number | null>(null);
  const [searchEpisodeTitle, setSearchEpisodeTitle] = useState<string | null>(
    null,
  );
  const [viewMode, setViewMode] = useState<"grid" | "table">("grid");
  const [editData, setEditData] = useState({
    title: "",
    year: "",
    metadata: {} as Record<string, unknown>,
  });

  const { data: media, isLoading, error } = useMediaItem(id!);
  const { data: files, isLoading: filesLoading } = useMediaFiles(id!);
  const { data: children, isLoading: childrenLoading } = useMediaList(
    id ? { parentId: Number(id) } : {},
  );
  const { data: parentMedia } = useMediaItem(
    media?.parent_id ? String(media.parent_id) : "",
  );
  const { data: grandparentMedia } = useMediaItem(
    parentMedia?.parent_id ? String(parentMedia.parent_id) : "",
  );
  const { data: downloadsData } = useDownloads();
  const updateMedia = useUpdateMedia(id!);
  const deleteFile = useDeleteMediaFile();
  const deleteMediaItem = useDeleteMediaItem();

  // Quality profile management
  const mediaIdNum = id ? parseInt(id) : 0;
  const { data: qualityProfiles } = useQualityProfiles();
  const { data: mediaQuality } = useMediaQuality(mediaIdNum);
  const { data: upgradeHistory } = useQualityUpgradeHistory(mediaIdNum);
  const assignProfile = useAssignProfileToMedia(mediaIdNum);

  // Monitoring
  const { data: monitoringRule } = useMonitoringRuleByMedia(mediaIdNum);
  const createMonitoring = useCreateMonitoringRule();
  const updateMonitoring = useUpdateMonitoringRule(monitoringRule?.id || 0);
  const deleteMonitoring = useDeleteMonitoringRule();

  // For TV seasons, fetch TMDB episode data
  const seasonMetadata =
    media?.kind === "tv_season" && media.metadata
      ? (media.metadata as Record<string, unknown>)
      : null;
  const seasonNumber = seasonMetadata?.season_number as number | null;
  const parentTmdbId =
    parentMedia?.metadata && typeof parentMedia.metadata === "object"
      ? ((parentMedia.metadata as Record<string, unknown>).tmdb_id as
          | string
          | null)
      : null;

  const { data: tmdbSeasonData } = useTVSeasonDetails(
    parentTmdbId ? parseInt(parentTmdbId) : null,
    seasonNumber,
  );

  // Map TMDB episodes to their status
  const episodesWithStatus = useMemo(() => {
    if (!tmdbSeasonData?.episodes || media?.kind !== "tv_season") return [];

    const existingEpisodes = new Map(
      (children?.items || []).map((item) => {
        const epNum =
          item.metadata && typeof item.metadata === "object"
            ? ((item.metadata as Record<string, unknown>).episode as number)
            : null;
        return [epNum, item];
      }),
    );

    // Create a set of media IDs that are currently being downloaded
    // Convert to numbers for consistent comparison
    const downloadingMediaIds = new Set(
      (downloadsData?.downloads || [])
        .filter(
          (d) =>
            d.status === "queued" ||
            d.status === "downloading" ||
            d.status === "processing",
        )
        .map((d) => {
          const mediaId = d.metadata?.media_id;
          if (mediaId === undefined || mediaId === null) return null;
          return typeof mediaId === "number" ? mediaId : Number(mediaId);
        })
        .filter((id): id is number => id !== null && !isNaN(id)),
    );

    return (tmdbSeasonData.episodes as any[]).map((ep: any) => {
      const existingEpisode = existingEpisodes.get(ep.episode_number);
      const hasFiles =
        existingEpisode && files
          ? files.some((f) => f.media_item_id === existingEpisode.id)
          : false;
      const isDownloading =
        existingEpisode &&
        downloadingMediaIds.has(existingEpisode.id as number);

      return {
        episode_number: ep.episode_number,
        name: ep.name,
        air_date: ep.air_date,
        runtime: ep.runtime,
        overview: ep.overview,
        still_path: ep.still_path,
        vote_average: ep.vote_average,
        existingEpisode,
        status: existingEpisode
          ? isDownloading
            ? "downloading"
            : hasFiles
              ? "available"
              : "missing"
          : "not_added",
      };
    });
  }, [tmdbSeasonData, children, media, files, downloadsData]);

  const handleEdit = () => {
    if (media) {
      setEditData({
        title: media.title,
        year: media.year?.toString() || "",
        metadata: (media.metadata as Record<string, unknown>) || {},
      });
      setIsEditOpen(true);
    }
  };

  const handleSave = async () => {
    try {
      await updateMedia.mutateAsync({
        title: editData.title,
        year: editData.year ? parseInt(editData.year) : null,
        metadata: editData.metadata,
      });
      setIsEditOpen(false);
    } catch (error) {
      console.error("Failed to update media:", error);
    }
  };

  const handleMetadataChange = (key: string, value: unknown) => {
    setEditData((prev) => ({
      ...prev,
      metadata: {
        ...prev.metadata,
        [key]: value,
      },
    }));
  };

  const handleDeleteMetadataField = (key: string) => {
    setEditData((prev) => {
      const newMetadata = { ...prev.metadata };
      delete newMetadata[key];
      return {
        ...prev,
        metadata: newMetadata,
      };
    });
  };

  const handleAddMetadataField = () => {
    const key = prompt("Enter metadata field name:");
    if (key && key.trim()) {
      handleMetadataChange(key.trim(), "");
    }
  };

  const handleDeleteFile = (fileId: number) => {
    setFileToDelete(fileId);
    setIsFileDeleteOpen(true);
  };

  const confirmDeleteFile = async (deletePhysical: boolean) => {
    if (fileToDelete === null) return;

    try {
      await deleteFile.mutateAsync({ fileId: fileToDelete, deletePhysical });
      setIsFileDeleteOpen(false);
      setFileToDelete(null);
    } catch (error) {
      console.error("Failed to delete file:", error);
      alert("Failed to delete file");
    }
  };

  const handleDeleteMedia = () => {
    setIsDeleteOpen(true);
    setDeleteConfirmed(false);
  };

  const confirmDeleteMedia = async (deleteFiles: boolean) => {
    if (!deleteConfirmed) {
      alert("Please confirm deletion by checking the box");
      return;
    }

    try {
      await deleteMediaItem.mutateAsync({ mediaId: Number(id), deleteFiles });
      navigate("/media");
    } catch (error) {
      console.error("Failed to delete media item:", error);
      alert("Failed to delete media item");
    }
  };

  const handleSelectRelease = (release: IndexerRelease) => {
    // Download is now handled by InteractiveSearchDialog
    // This callback is optional and just logs the selection
    console.log("Selected release:", release);
  };

  const handleAssignProfile = async (profileId: string) => {
    try {
      await assignProfile.mutateAsync({ profile_id: parseInt(profileId) });
    } catch (error) {
      console.error("Failed to assign quality profile:", error);
    }
  };

  const handleToggleMonitoring = async () => {
    try {
      if (monitoringRule) {
        await updateMonitoring.mutateAsync({
          enabled: !monitoringRule.enabled,
        });
      } else {
        // Create new monitoring rule with defaults
        try {
          await createMonitoring.mutateAsync({
            media_item_id: mediaIdNum,
            enabled: true,
            quality_profile_id: mediaQuality?.profile_id || undefined,
            monitor_mode: "all",
            search_on_add: true,
            automatic_search: true,
            backlog_search: true,
            prefer_season_packs: false,
            minimum_seeders: 1,
            tags: [],
            search_interval_minutes: 60,
          });
        } catch (createError: any) {
          // If rule already exists (race condition), reload to fetch it
          if (
            createError?.message?.includes("duplicate") ||
            createError?.message?.includes("already exists")
          ) {
            window.location.reload();
            return;
          }
          throw createError;
        }
      }
    } catch (error) {
      console.error("Failed to toggle monitoring:", error);
      alert("Failed to update monitoring");
    }
  };

  const formatFileSize = (bytes: number | null) => {
    if (bytes === null) return "Unknown";
    const units = ["B", "KB", "MB", "GB", "TB"];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  const formatRuntime = (minutes: number) => {
    const hours = Math.floor(minutes / 60);
    const mins = minutes % 60;
    if (hours > 0) {
      return `${hours}h ${mins}m`;
    }
    return `${mins}m`;
  };

  const getBackdropUrl = () => {
    if (!media?.metadata || typeof media.metadata !== "object") {
      // For TV seasons without metadata, try parent backdrop
      if (
        media?.kind === "tv_season" &&
        parentMedia?.metadata &&
        typeof parentMedia.metadata === "object"
      ) {
        const parentMetadata = parentMedia.metadata as Record<string, unknown>;
        return parentMetadata.backdrop_url as string | null;
      }
      return null;
    }

    const metadata = media.metadata as Record<string, unknown>;

    // For episodes, prefer still_url, fallback to backdrop_url
    if (media.kind === "tv_episode" && metadata.still_url) {
      return metadata.still_url as string;
    }

    // Check for backdrop_url
    if (metadata.backdrop_url) {
      return metadata.backdrop_url as string;
    }

    // For TV seasons without backdrop, use parent's backdrop
    if (
      media.kind === "tv_season" &&
      parentMedia?.metadata &&
      typeof parentMedia.metadata === "object"
    ) {
      const parentMetadata = parentMedia.metadata as Record<string, unknown>;
      return parentMetadata.backdrop_url as string | null;
    }

    return null;
  };

  const getRating = () => {
    if (!media?.metadata || typeof media.metadata !== "object") return null;
    const metadata = media.metadata as Record<string, unknown>;
    return metadata.rating as number | undefined;
  };

  const getRuntime = () => {
    if (!media?.metadata || typeof media.metadata !== "object") return null;
    const metadata = media.metadata as Record<string, unknown>;
    return metadata.runtime as number | undefined;
  };

  const getAirDate = () => {
    if (!media?.metadata || typeof media.metadata !== "object") return null;
    const metadata = media.metadata as Record<string, unknown>;
    return (metadata.air_date ||
      metadata.first_air_date ||
      metadata.release_date) as string | undefined;
  };

  const getDescription = () => {
    if (!media?.metadata || typeof media.metadata !== "object") return null;
    const metadata = media.metadata as Record<string, unknown>;
    return metadata.description as string | undefined;
  };

  const getGenres = () => {
    if (!media?.metadata || typeof media.metadata !== "object") return null;
    const metadata = media.metadata as Record<string, unknown>;
    const genres = metadata.genres;
    if (Array.isArray(genres)) {
      // Handle both string array and object array {id, name}
      return genres.map((genre) => {
        if (typeof genre === "string") {
          return genre;
        } else if (
          typeof genre === "object" &&
          genre !== null &&
          "name" in genre
        ) {
          return (genre as { name: string }).name;
        }
        return String(genre);
      });
    }
    return null;
  };

  if (error) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" onClick={() => navigate(-1)}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-destructive">
              Failed to load media item
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!media) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" onClick={() => navigate(-1)}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">
              Media item not found
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const backdropUrl = getBackdropUrl();
  const rating = getRating();
  const runtime = getRuntime();
  const airDate = getAirDate();
  const description = getDescription();
  const genres = getGenres();

  return (
    <div className="space-y-6">
      {/* Backdrop Header */}
      {backdropUrl ? (
        <div className="relative -mx-6 -mt-6">
          <div className="relative h-[500px] overflow-hidden">
            {/* Backdrop Image */}
            <img
              src={backdropUrl}
              alt={media.title}
              className="w-full h-full object-cover"
            />

            {/* Gradient Overlay */}
            <div className="absolute inset-0 bg-gradient-to-t from-background via-background/80 to-transparent" />

            {/* Content Overlay */}
            <div className="absolute inset-0 flex flex-col justify-end">
              <div className="px-6 pb-8 space-y-4">
                {/* Navigation Buttons */}
                <div className="flex items-center justify-between">
                  <Button variant="secondary" onClick={() => navigate(-1)}>
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Back
                  </Button>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      onClick={() => setIsSearchOpen(true)}
                    >
                      <Search className="mr-2 h-4 w-4" />
                      Search Releases
                    </Button>
                    <Button variant="secondary" onClick={handleEdit}>
                      <Pencil className="mr-2 h-4 w-4" />
                      Edit
                    </Button>
                    <Button variant="destructive" onClick={handleDeleteMedia}>
                      <Trash2 className="mr-2 h-4 w-4" />
                      Delete
                    </Button>
                  </div>
                </div>

                {/* Title and Metadata */}
                <div className="space-y-3">
                  <div className="flex items-center gap-3 flex-wrap">
                    <h1 className="text-4xl font-bold flex items-center gap-2 dark:text-white drop-shadow-lg">
                      {/* For seasons, show series name */}
                      {media.kind === "tv_season" && parentMedia && (
                        <>
                          <button
                            onClick={() => navigate(`/media/${parentMedia.id}`)}
                            className="dark:text-white/80 dark:hover:text-white hover:text-gray-500 transition-colors"
                          >
                            {parentMedia.title}
                          </button>
                          <span className="dark:text-white/80">/</span>
                        </>
                      )}
                      {/* For episodes, show series name and season */}
                      {media.kind === "tv_episode" &&
                        grandparentMedia &&
                        parentMedia && (
                          <>
                            <button
                              onClick={() =>
                                navigate(`/media/${grandparentMedia.id}`)
                              }
                              className="dark:text-white/80 dark:hover:text-white text-gray-700 hover:text-gray-500 transition-colors"
                            >
                              {grandparentMedia.title}
                            </button>
                            <span className="dark:text-white/80 text-gray-700">
                              /
                            </span>
                            <button
                              onClick={() =>
                                navigate(`/media/${parentMedia.id}`)
                              }
                              className="dark:text-white/80 dark:hover:text-white text-gray-700 hover:text-gray-500 transition-colors"
                            >
                              {parentMedia.title}
                            </button>
                            <span className="dark:text-white/80 text-gray-700">
                              /
                            </span>
                          </>
                        )}
                      <span>
                        {media.kind === "tv_episode" &&
                        media.metadata &&
                        typeof media.metadata === "object" &&
                        (media.metadata as Record<string, unknown>).episode ? (
                          <span className="dark:text-white/80 text-gray-700">
                            E
                            {String(
                              (media.metadata as Record<string, unknown>)
                                .episode,
                            )}{" "}
                          </span>
                        ) : null}
                        {media.title}
                      </span>
                    </h1>
                    <MediaKindBadge kind={media.kind} />
                  </div>

                  {/* Metadata Row */}
                  <div className="flex items-center gap-4 flex-wrap dark:text-white/90 drop-shadow">
                    {media.year && (
                      <span className="text-lg font-medium">{media.year}</span>
                    )}
                    {rating && (
                      <div className="flex items-center gap-1">
                        <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                        <span className="font-medium">{rating.toFixed(1)}</span>
                      </div>
                    )}
                    {runtime && (
                      <div className="flex items-center gap-1">
                        <Clock className="h-4 w-4" />
                        <span>{formatRuntime(runtime)}</span>
                      </div>
                    )}
                    {airDate && (
                      <div className="flex items-center gap-1">
                        <Calendar className="h-4 w-4" />
                        <span>{new Date(airDate).toLocaleDateString()}</span>
                      </div>
                    )}
                  </div>

                  {/* Genres */}
                  {genres && genres.length > 0 && (
                    <div className="flex items-center gap-2 flex-wrap">
                      {genres.map((genre) => (
                        <Badge
                          key={genre}
                          variant="secondary"
                          className="dark:bg-white/20 dark:text-white dark:border-white/30 dark:hover:bg-white/30 bg-slate-400 text-gray-800 border-slate-400 hover:bg-slate-300"
                        >
                          {genre}
                        </Badge>
                      ))}
                    </div>
                  )}

                  {/* Description */}
                  {description && (
                    <p className="dark:text-white/90 max-w-4xl text-lg drop-shadow line-clamp-3">
                      {description}
                    </p>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <>
          {/* Fallback Header (no backdrop) */}
          <div className="flex items-center justify-between">
            <Button variant="ghost" onClick={() => navigate(-1)}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back
            </Button>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setIsSearchOpen(true)}>
                <Search className="mr-2 h-4 w-4" />
                Search Releases
              </Button>
              <Button onClick={handleEdit}>
                <Pencil className="mr-2 h-4 w-4" />
                Edit
              </Button>
              <Button variant="destructive" onClick={handleDeleteMedia}>
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </Button>
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center gap-3 flex-wrap">
              <h1 className="text-3xl font-bold flex items-center gap-2">
                {/* For seasons, show series name */}
                {media.kind === "tv_season" && parentMedia && (
                  <>
                    <button
                      onClick={() => navigate(`/media/${parentMedia.id}`)}
                      className="text-muted-foreground hover:text-foreground transition-colors"
                    >
                      {parentMedia.title}
                    </button>
                    <span className="text-muted-foreground">/</span>
                  </>
                )}
                {/* For episodes, show series name and season */}
                {media.kind === "tv_episode" &&
                  grandparentMedia &&
                  parentMedia && (
                    <>
                      <button
                        onClick={() =>
                          navigate(`/media/${grandparentMedia.id}`)
                        }
                        className="text-muted-foreground hover:text-foreground transition-colors"
                      >
                        {grandparentMedia.title}
                      </button>
                      <span className="text-muted-foreground">/</span>
                      <button
                        onClick={() => navigate(`/media/${parentMedia.id}`)}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                      >
                        {parentMedia.title}
                      </button>
                      <span className="text-muted-foreground">/</span>
                    </>
                  )}
                <span>
                  {media.kind === "tv_episode" &&
                  media.metadata &&
                  typeof media.metadata === "object" &&
                  (media.metadata as Record<string, unknown>).episode ? (
                    <span className="text-muted-foreground">
                      E
                      {String(
                        (media.metadata as Record<string, unknown>).episode,
                      )}{" "}
                    </span>
                  ) : null}
                  {media.title}
                </span>
              </h1>
              <MediaKindBadge kind={media.kind} />
            </div>
            {media.year && (
              <p className="text-lg text-muted-foreground">{media.year}</p>
            )}
            {description && (
              <p className="text-muted-foreground mt-2">{description}</p>
            )}
          </div>
        </>
      )}

      {/* Episodes Section for TV Seasons */}
      {media.kind === "tv_season" && episodesWithStatus.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Episodes</CardTitle>
            <CardDescription>
              {episodesWithStatus.length}{" "}
              {episodesWithStatus.length === 1 ? "episode" : "episodes"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">#</TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead className="w-32">Air Date</TableHead>
                  <TableHead className="w-24">Runtime</TableHead>
                  <TableHead className="w-24">Rating</TableHead>
                  <TableHead className="w-32">Status</TableHead>
                  <TableHead className="w-16">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {episodesWithStatus.map((episode) => (
                  <TableRow
                    key={episode.episode_number}
                    className={episode.existingEpisode ? "cursor-pointer" : ""}
                    onClick={() => {
                      if (episode.existingEpisode) {
                        navigate(`/media/${episode.existingEpisode.id}`);
                      }
                    }}
                  >
                    <TableCell className="font-medium">
                      {episode.episode_number}
                    </TableCell>
                    <TableCell>
                      <div>
                        <p className="font-medium">{episode.name}</p>
                        {episode.overview && (
                          <p className="text-xs text-muted-foreground line-clamp-2 mt-1">
                            {episode.overview}
                          </p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {episode.air_date
                        ? new Date(episode.air_date).toLocaleDateString()
                        : "-"}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {episode.runtime ? `${episode.runtime}m` : "-"}
                    </TableCell>
                    <TableCell>
                      {episode.vote_average > 0 && (
                        <div className="flex items-center gap-1">
                          <Star className="h-3 w-3 fill-yellow-400 text-yellow-400" />
                          <span className="text-sm">
                            {episode.vote_average.toFixed(1)}
                          </span>
                        </div>
                      )}
                    </TableCell>
                    <TableCell>
                      {episode.status === "available" ? (
                        <Badge variant="default" className="gap-1">
                          <CheckCircle className="h-3 w-3" />
                          Available
                        </Badge>
                      ) : episode.status === "missing" ? (
                        <Badge variant="secondary" className="gap-1">
                          <XCircle className="h-3 w-3" />
                          Missing
                        </Badge>
                      ) : episode.status === "downloading" ? (
                        <Badge variant="outline" className="gap-1">
                          <Download className="h-3 w-3" />
                          Downloading
                        </Badge>
                      ) : (
                        <Badge variant="outline" className="gap-1">
                          <XCircle className="h-3 w-3" />
                          Not Added
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {episode.existingEpisode ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0"
                          onClick={(e) => {
                            e.stopPropagation();
                            setSearchEpisodeId(
                              episode.existingEpisode!.id as number,
                            );
                            setSearchEpisodeTitle(episode.name);
                            setIsSearchOpen(true);
                          }}
                        >
                          <Search className="h-4 w-4" />
                        </Button>
                      ) : (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0"
                          disabled
                        >
                          <Search className="h-4 w-4 opacity-30" />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Children Section for non-season kinds */}
      {media.kind !== "tv_season" &&
        (childrenLoading ||
          (children && children.items && children.items.length > 0)) && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>
                    {media.kind === "tv_series"
                      ? "Seasons"
                      : media.kind === "music_artist"
                        ? "Albums"
                        : media.kind === "music_album"
                          ? "Tracks"
                          : "Children"}
                  </CardTitle>
                  {children && (
                    <CardDescription>
                      {children.total} {children.total === 1 ? "item" : "items"}
                    </CardDescription>
                  )}
                </div>
                {children && children.items && children.items.length > 0 && (
                  <div className="flex gap-2">
                    <Button
                      variant={viewMode === "grid" ? "default" : "outline"}
                      size="sm"
                      onClick={() => setViewMode("grid")}
                    >
                      <LayoutGrid className="h-4 w-4 mr-2" />
                      Grid
                    </Button>
                    <Button
                      variant={viewMode === "table" ? "default" : "outline"}
                      size="sm"
                      onClick={() => setViewMode("table")}
                    >
                      <TableIcon className="h-4 w-4 mr-2" />
                      Table
                    </Button>
                  </div>
                )}
              </div>
            </CardHeader>
            <CardContent>
              {childrenLoading ? (
                <div className="flex items-center justify-center py-12">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              ) : children && children.items && children.items.length > 0 ? (
                <>
                  {viewMode === "grid" && <MediaGrid items={children.items} />}
                  {viewMode === "table" && (
                    <MediaTable items={children.items} />
                  )}
                </>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No children found
                </p>
              )}
            </CardContent>
          </Card>
        )}

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Details</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <Label className="text-muted-foreground">ID</Label>
              <p className="font-mono text-sm">{media.id}</p>
            </div>
            <div>
              <Label className="text-muted-foreground">Sort Title</Label>
              <p>{media.sort_title}</p>
            </div>
            <div>
              <Label className="text-muted-foreground">Created</Label>
              <p className="text-sm">{formatDate(media.created_at)}</p>
            </div>
            <div>
              <Label className="text-muted-foreground">Updated</Label>
              <p className="text-sm">{formatDate(media.updated_at)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Files</CardTitle>
            <CardDescription>Associated media files</CardDescription>
          </CardHeader>
          <CardContent>
            {filesLoading ? (
              <div className="flex items-center justify-center py-4">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : files && files.length > 0 ? (
              <div className="space-y-2">
                {files.map((file) => (
                  <div
                    key={file.id}
                    className="flex items-start justify-between p-3 rounded-lg bg-muted group"
                  >
                    <div className="flex-1 min-w-0 space-y-1">
                      <div className="flex items-center gap-2">
                        <File className="h-4 w-4 text-muted-foreground shrink-0" />
                        <p
                          className="text-sm font-mono truncate"
                          title={file.path}
                        >
                          {file.path.split("/").pop()}
                        </p>
                      </div>
                      <div className="flex items-center gap-3 text-xs text-muted-foreground">
                        <span>{formatFileSize(file.size)}</span>
                        <span>•</span>
                        <span className="font-mono truncate" title={file.path}>
                          {file.path}
                        </span>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 w-8 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                      onClick={() => handleDeleteFile(file.id)}
                    >
                      <X className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No files</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Quality Profile Section */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Award className="h-5 w-5" />
                  Quality Profile
                </CardTitle>
                <CardDescription>
                  Configure quality preferences and monitoring
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="quality-profile">Assigned Profile</Label>
              <Select
                value={mediaQuality?.profile_id?.toString() || ""}
                onValueChange={handleAssignProfile}
                disabled={assignProfile.isPending}
              >
                <SelectTrigger id="quality-profile">
                  <SelectValue placeholder="Select quality profile" />
                </SelectTrigger>
                <SelectContent>
                  {qualityProfiles?.map((profile) => (
                    <SelectItem key={profile.id} value={profile.id.toString()}>
                      {profile.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {!mediaQuality && (
                <p className="text-xs text-muted-foreground">
                  No quality profile assigned yet. Select one to enable quality
                  monitoring.
                </p>
              )}
            </div>

            {mediaQuality && mediaQuality.quality && (
              <>
                <div className="p-3 bg-muted rounded-lg space-y-2">
                  <Label className="text-sm text-muted-foreground">
                    Current Quality
                  </Label>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">
                      {mediaQuality.quality.title}
                    </Badge>
                    {mediaQuality.quality.resolution && (
                      <span className="text-sm text-muted-foreground">
                        {mediaQuality.quality.resolution}p
                      </span>
                    )}
                    {mediaQuality.quality.source && (
                      <span className="text-sm text-muted-foreground">
                        · {mediaQuality.quality.source}
                      </span>
                    )}
                  </div>
                  {(mediaQuality.codec_video || mediaQuality.codec_audio) && (
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      {mediaQuality.codec_video && (
                        <span>Video: {mediaQuality.codec_video}</span>
                      )}
                      {mediaQuality.codec_audio && (
                        <span>Audio: {mediaQuality.codec_audio}</span>
                      )}
                    </div>
                  )}
                </div>

                <div className="flex items-center gap-4">
                  {mediaQuality.cutoff_met ? (
                    <div className="flex items-center gap-2 text-sm">
                      <CheckCircle className="h-4 w-4 text-green-600" />
                      <span className="text-green-600">Cutoff met</span>
                    </div>
                  ) : mediaQuality.upgrade_allowed ? (
                    <div className="flex items-center gap-2 text-sm">
                      <TrendingUp className="h-4 w-4 text-blue-600" />
                      <span className="text-blue-600">Upgrades allowed</span>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2 text-sm">
                      <AlertCircle className="h-4 w-4 text-muted-foreground" />
                      <span className="text-muted-foreground">
                        Upgrades disabled
                      </span>
                    </div>
                  )}
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Quality Upgrade History</CardTitle>
            <CardDescription>
              Track quality improvements over time
            </CardDescription>
          </CardHeader>
          <CardContent>
            {upgradeHistory && upgradeHistory.length > 0 ? (
              <div className="space-y-3">
                {upgradeHistory.slice(0, 5).map((upgrade) => (
                  <div
                    key={upgrade.id}
                    className="p-3 bg-muted rounded-lg space-y-1"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        {upgrade.old_quality && (
                          <Badge variant="outline" className="text-xs">
                            {upgrade.old_quality.title}
                          </Badge>
                        )}
                        <span className="text-muted-foreground">→</span>
                        {upgrade.new_quality && (
                          <Badge variant="default" className="text-xs">
                            {upgrade.new_quality.title}
                          </Badge>
                        )}
                      </div>
                    </div>
                    {upgrade.reason && (
                      <p className="text-xs text-muted-foreground">
                        {upgrade.reason}
                      </p>
                    )}
                    <p className="text-xs text-muted-foreground">
                      {new Date(upgrade.created_at).toLocaleString()}
                    </p>
                  </div>
                ))}
                {upgradeHistory.length > 5 && (
                  <p className="text-xs text-muted-foreground text-center">
                    + {upgradeHistory.length - 5} more upgrades
                  </p>
                )}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                No quality upgrades recorded
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Monitoring Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Activity className="h-5 w-5" />
                Monitoring & Automation
              </CardTitle>
              <CardDescription>
                Automatically search and download new releases
              </CardDescription>
            </div>
            <Button
              onClick={handleToggleMonitoring}
              variant={monitoringRule?.enabled ? "default" : "outline"}
              disabled={
                createMonitoring.isPending || updateMonitoring.isPending
              }
            >
              {createMonitoring.isPending || updateMonitoring.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {monitoringRule ? "Updating..." : "Enabling..."}
                </>
              ) : monitoringRule?.enabled ? (
                <>
                  <Eye className="mr-2 h-4 w-4" />
                  Monitored
                </>
              ) : (
                <>
                  <EyeOff className="mr-2 h-4 w-4" />
                  {monitoringRule ? "Disabled" : "Enable Monitoring"}
                </>
              )}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {monitoringRule ? (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <Label className="text-sm text-muted-foreground">
                    Status
                  </Label>
                  <div className="flex items-center gap-2">
                    {monitoringRule.enabled ? (
                      <>
                        <CheckCircle className="h-4 w-4 text-green-600" />
                        <span className="text-sm">Active</span>
                      </>
                    ) : (
                      <>
                        <XCircle className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm text-muted-foreground">
                          Inactive
                        </span>
                      </>
                    )}
                  </div>
                </div>
                <div className="space-y-1">
                  <Label className="text-sm text-muted-foreground">
                    Monitor Mode
                  </Label>
                  <Badge variant="secondary">
                    {monitoringRule.monitor_mode}
                  </Badge>
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4 p-3 bg-muted rounded-lg">
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">
                    Searches
                  </Label>
                  <p className="text-lg font-semibold">
                    {monitoringRule.search_count}
                  </p>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Found</Label>
                  <p className="text-lg font-semibold">
                    {monitoringRule.items_found_count}
                  </p>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">
                    Grabbed
                  </Label>
                  <p className="text-lg font-semibold">
                    {monitoringRule.items_grabbed_count}
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <div className="flex items-center gap-2 text-sm">
                  {monitoringRule.automatic_search && (
                    <Badge variant="outline" className="text-xs">
                      Auto Search
                    </Badge>
                  )}
                  {monitoringRule.backlog_search && (
                    <Badge variant="outline" className="text-xs">
                      Backlog
                    </Badge>
                  )}
                  {monitoringRule.prefer_season_packs && (
                    <Badge variant="outline" className="text-xs">
                      Season Packs
                    </Badge>
                  )}
                </div>
                {monitoringRule.last_search_at && (
                  <p className="text-xs text-muted-foreground">
                    Last search:{" "}
                    {new Date(monitoringRule.last_search_at).toLocaleString()}
                  </p>
                )}
                {monitoringRule.next_search_at && monitoringRule.enabled && (
                  <p className="text-xs text-muted-foreground">
                    Next search:{" "}
                    {new Date(monitoringRule.next_search_at).toLocaleString()}
                  </p>
                )}
              </div>
            </div>
          ) : (
            <div className="text-center py-4">
              <p className="text-sm text-muted-foreground mb-3">
                This media item is not being monitored. Enable monitoring to
                automatically search for and download new releases.
              </p>
              <Button
                onClick={handleToggleMonitoring}
                disabled={createMonitoring.isPending}
              >
                {createMonitoring.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Enabling...
                  </>
                ) : (
                  <>
                    <Activity className="mr-2 h-4 w-4" />
                    Enable Monitoring
                  </>
                )}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Edit Dialog */}
      <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Media Item</DialogTitle>
            <DialogDescription>
              Update information and metadata for this media item
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="title">Title</Label>
              <Input
                id="title"
                value={editData.title}
                onChange={(e) =>
                  setEditData((prev) => ({ ...prev, title: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="year">Year</Label>
              <Input
                id="year"
                type="number"
                value={editData.year}
                onChange={(e) =>
                  setEditData((prev) => ({ ...prev, year: e.target.value }))
                }
              />
            </div>

            {/* Metadata Fields */}
            <div className="space-y-3 pt-4 border-t">
              <div className="flex items-center justify-between">
                <Label className="text-base font-semibold">Metadata</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleAddMetadataField}
                >
                  <Plus className="h-4 w-4 mr-1" />
                  Add Field
                </Button>
              </div>

              {Object.entries(editData.metadata).map(([key, value]) => {
                const isTextarea =
                  typeof value === "string" && value.length > 100;
                const isNumber = typeof value === "number";
                const isBoolean = typeof value === "boolean";
                const isObject = typeof value === "object" && value !== null;

                return (
                  <div key={key} className="grid gap-2 p-3 border rounded-lg">
                    <div className="flex items-center justify-between">
                      <Label
                        htmlFor={`metadata-${key}`}
                        className="font-medium"
                      >
                        {key}
                      </Label>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDeleteMetadataField(key)}
                        className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    </div>

                    {isObject ? (
                      <Textarea
                        id={`metadata-${key}`}
                        value={JSON.stringify(value, null, 2)}
                        onChange={(e) => {
                          try {
                            const parsed = JSON.parse(e.target.value);
                            handleMetadataChange(key, parsed);
                          } catch {
                            // Keep editing invalid JSON
                            handleMetadataChange(key, e.target.value);
                          }
                        }}
                        rows={4}
                        className="font-mono text-sm"
                      />
                    ) : isBoolean ? (
                      <div className="flex items-center space-x-2">
                        <input
                          id={`metadata-${key}`}
                          type="checkbox"
                          checked={value}
                          onChange={(e) =>
                            handleMetadataChange(key, e.target.checked)
                          }
                          className="h-4 w-4 rounded border-input"
                        />
                        <Label htmlFor={`metadata-${key}`} className="text-sm">
                          {value ? "True" : "False"}
                        </Label>
                      </div>
                    ) : isTextarea ? (
                      <Textarea
                        id={`metadata-${key}`}
                        value={String(value)}
                        onChange={(e) =>
                          handleMetadataChange(key, e.target.value)
                        }
                        rows={4}
                      />
                    ) : (
                      <Input
                        id={`metadata-${key}`}
                        type={isNumber ? "number" : "text"}
                        value={String(value)}
                        onChange={(e) => {
                          const newValue = isNumber
                            ? parseFloat(e.target.value)
                            : e.target.value;
                          handleMetadataChange(key, newValue);
                        }}
                      />
                    )}
                  </div>
                );
              })}

              {Object.keys(editData.metadata).length === 0 && (
                <p className="text-sm text-muted-foreground text-center py-4">
                  No metadata fields. Click "Add Field" to create one.
                </p>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsEditOpen(false)}
              disabled={updateMedia.isPending}
            >
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={updateMedia.isPending}>
              {updateMedia.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Media Dialog */}
      <Dialog open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Media Item</DialogTitle>
            <DialogDescription>
              This action cannot be undone. This will permanently delete the
              media item from the database.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="confirm-delete"
                checked={deleteConfirmed}
                onCheckedChange={(checked) =>
                  setDeleteConfirmed(checked as boolean)
                }
              />
              <label
                htmlFor="confirm-delete"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                I understand this action cannot be undone
              </label>
            </div>
            {files && files.length > 0 && (
              <div className="rounded-lg border border-border p-4 space-y-2">
                <p className="text-sm font-medium">
                  This media item has {files.length} associated file(s)
                </p>
                <p className="text-sm text-muted-foreground">
                  Choose whether to keep or delete the physical files:
                </p>
              </div>
            )}
          </div>
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={() => {
                setIsDeleteOpen(false);
                setDeleteConfirmed(false);
              }}
              disabled={deleteMediaItem.isPending}
              className="w-full sm:w-auto"
            >
              Cancel
            </Button>
            <Button
              variant="secondary"
              onClick={() => confirmDeleteMedia(false)}
              disabled={deleteMediaItem.isPending || !deleteConfirmed}
              className="w-full sm:w-auto"
            >
              {deleteMediaItem.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <HardDrive className="mr-2 h-4 w-4" />
              )}
              Delete (Keep Files)
            </Button>
            <Button
              variant="destructive"
              onClick={() => confirmDeleteMedia(true)}
              disabled={deleteMediaItem.isPending || !deleteConfirmed}
              className="w-full sm:w-auto"
            >
              {deleteMediaItem.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Trash2 className="mr-2 h-4 w-4" />
              )}
              Delete (With Files)
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete File Dialog */}
      <Dialog open={isFileDeleteOpen} onOpenChange={setIsFileDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete File</DialogTitle>
            <DialogDescription>
              Choose whether to remove just the database entry or also delete
              the physical file.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={() => {
                setIsFileDeleteOpen(false);
                setFileToDelete(null);
              }}
              disabled={deleteFile.isPending}
              className="w-full sm:w-auto"
            >
              Cancel
            </Button>
            <Button
              variant="secondary"
              onClick={() => confirmDeleteFile(false)}
              disabled={deleteFile.isPending}
              className="w-full sm:w-auto"
            >
              {deleteFile.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <HardDrive className="mr-2 h-4 w-4" />
              )}
              Remove Entry Only
            </Button>
            <Button
              variant="destructive"
              onClick={() => confirmDeleteFile(true)}
              disabled={deleteFile.isPending}
              className="w-full sm:w-auto"
            >
              {deleteFile.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Trash2 className="mr-2 h-4 w-4" />
              )}
              Delete from Disk
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Interactive Search Dialog */}
      {media && (
        <InteractiveSearchDialog
          mediaId={searchEpisodeId || id!}
          mediaTitle={
            searchEpisodeId ? searchEpisodeTitle || media.title : media.title
          }
          mediaKind={searchEpisodeId ? "tv_episode" : media.kind}
          open={isSearchOpen}
          onOpenChange={(open) => {
            setIsSearchOpen(open);
            if (!open) {
              setSearchEpisodeId(null);
              setSearchEpisodeTitle(null);
            }
          }}
          onSelectRelease={handleSelectRelease}
        />
      )}
    </div>
  );
}
