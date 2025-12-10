import { useState, useEffect } from "react";
import { useSearchParams, Link } from "react-router-dom";
import { useMediaList } from "@/lib/api/media";
import {
  useSearchMovies,
  useSearchTV,
  useAddToLibrary,
  useTVDetails,
} from "@/lib/api/tmdb";
import { useTMDBInLibrary } from "@/lib/api/media";
import type { TMDBSearchResult } from "@/lib/api/tmdb";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { MediaGrid } from "@/components/media/MediaGrid";
import {
  Loader2,
  Film,
  Tv,
  Plus,
  CheckCircle2,
  ExternalLink,
  Star,
  Calendar,
} from "lucide-react";
import { cn } from "@/lib/utils";

const TMDB_IMAGE_BASE_URL = "https://image.tmdb.org/t/p/w500";

export default function SearchPage() {
  const [searchParams] = useSearchParams();
  const query = searchParams.get("q") || "";
  const [debouncedQuery, setDebouncedQuery] = useState(query);
  const [addedItems, setAddedItems] = useState<Set<number>>(new Set());
  const [addingItem, setAddingItem] = useState<number | null>(null);
  const [showSeasonModal, setShowSeasonModal] = useState(false);
  const [selectedItem, setSelectedItem] = useState<TMDBSearchResult | null>(
    null,
  );
  const [selectedSeasons, setSelectedSeasons] = useState<number[]>([]);

  const addToLibrary = useAddToLibrary();

  // Fetch TV details when modal is open
  const tvDetails = useTVDetails(
    showSeasonModal && selectedItem && !selectedItem.title
      ? selectedItem.id
      : null,
  );

  // Debounce query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query);
    }, 300);
    return () => clearTimeout(timer);
  }, [query]);

  // Search library
  const libraryResults = useMediaList({
    q: debouncedQuery || undefined,
    limit: 50,
  });

  // Search TMDB (movies and TV)
  const tmdbMovies = useSearchMovies(debouncedQuery);
  const tmdbTV = useSearchTV(debouncedQuery);

  // Combine TMDB results
  const tmdbResults = [
    ...(tmdbMovies.data?.results || []),
    ...(tmdbTV.data?.results || []),
  ].sort((a, b) => b.vote_average - a.vote_average);

  // Get TMDB IDs to check library
  const tmdbIds = tmdbResults.map((item) => item.id);
  const { data: libraryCheck } = useTMDBInLibrary(tmdbIds);

  const handleAddToLibrary = async (
    item: TMDBSearchResult,
    seasonsToAdd?: number[],
  ) => {
    setAddingItem(item.id);
    try {
      const title = item.title || item.name || "";
      const year = item.release_date
        ? new Date(item.release_date).getFullYear()
        : item.first_air_date
          ? new Date(item.first_air_date).getFullYear()
          : undefined;

      const mediaType = item.title ? "movie" : "tv";

      await addToLibrary.mutateAsync({
        tmdb_id: item.id,
        media_type: mediaType,
        title,
        year,
        seasons: seasonsToAdd,
      });

      setAddedItems((prev) => new Set(prev).add(item.id));
    } catch (error) {
      console.error("Failed to add to library:", error);
    } finally {
      setAddingItem(null);
    }
  };

  const handleAddClick = (item: TMDBSearchResult) => {
    const mediaType = item.title ? "movie" : "tv";

    if (mediaType === "tv") {
      // Show season selection modal for TV shows
      setSelectedItem(item);
      setSelectedSeasons([]);
      setShowSeasonModal(true);
    } else {
      // Add movie directly
      handleAddToLibrary(item);
    }
  };

  const handleConfirmSeasons = async () => {
    if (selectedItem) {
      const itemToAdd = selectedItem;
      const seasonsToAdd =
        selectedSeasons.length > 0 ? selectedSeasons : undefined;

      setShowSeasonModal(false);
      setSelectedItem(null);

      await handleAddToLibrary(itemToAdd, seasonsToAdd);
    }
  };

  const toggleSeason = (seasonNumber: number) => {
    setSelectedSeasons((prev) =>
      prev.includes(seasonNumber)
        ? prev.filter((s) => s !== seasonNumber)
        : [...prev, seasonNumber],
    );
  };

  const toggleAllSeasons = () => {
    const seasons = tvDetails.data?.seasons || [];
    if (selectedSeasons.length === seasons.length) {
      setSelectedSeasons([]);
    } else {
      setSelectedSeasons(seasons.map((s: any) => s.season_number));
    }
  };

  const isAdded = (id: number) => addedItems.has(id);
  const isAdding = (id: number) => addingItem === id;
  const getLibraryId = (tmdbId: number): number | null => {
    return libraryCheck?.[tmdbId] || null;
  };
  const isInLibrary = (tmdbId: number): boolean => {
    return getLibraryId(tmdbId) !== null;
  };

  if (!debouncedQuery) {
    return (
      <div className="container mx-auto py-16 text-center">
        <p className="text-muted-foreground">
          Enter a search query to find media
        </p>
      </div>
    );
  }

  const isLoading =
    libraryResults.isLoading || tmdbMovies.isLoading || tmdbTV.isLoading;

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Search Results</h1>
        <p className="text-muted-foreground">
          Showing results for "{debouncedQuery}"
        </p>
      </div>

      {/* Library Results */}
      <Card>
        <CardHeader>
          <CardTitle>
            In Your Library ({libraryResults.data?.total || 0})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {libraryResults.isLoading && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
            </div>
          )}

          {libraryResults.data && libraryResults.data.items.length === 0 && (
            <p className="text-sm text-muted-foreground py-4">
              No items found in your library
            </p>
          )}

          {libraryResults.data && libraryResults.data.items.length > 0 && (
            <MediaGrid items={libraryResults.data.items} />
          )}
        </CardContent>
      </Card>

      {/* TMDB Results */}
      <Card>
        <CardHeader>
          <CardTitle>Browse & Add ({tmdbResults.length})</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
            </div>
          )}

          {!isLoading && tmdbResults.length === 0 && (
            <p className="text-sm text-muted-foreground py-4">
              No results found on TMDB
            </p>
          )}

          {tmdbResults.length > 0 && (
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
              {tmdbResults.map((item) => {
                const mediaType = item.title ? "movie" : "tv";
                return (
                  <Card
                    key={`${item.id}-${item.title || item.name}`}
                    className={cn(
                      "overflow-hidden hover:shadow-lg transition-shadow",
                      (isAdded(item.id) || isInLibrary(item.id)) &&
                        "opacity-60",
                    )}
                  >
                    <Link to={`/tmdb/${mediaType}/${item.id}`}>
                      {/* Poster */}
                      <div className="relative aspect-[2/3] bg-muted cursor-pointer">
                        {item.poster_path ? (
                          <img
                            src={`${TMDB_IMAGE_BASE_URL}${item.poster_path}`}
                            alt={item.title || item.name}
                            className="w-full h-full object-cover"
                          />
                        ) : (
                          <div className="w-full h-full flex items-center justify-center">
                            {item.title ? (
                              <Film className="h-12 w-12 text-muted-foreground" />
                            ) : (
                              <Tv className="h-12 w-12 text-muted-foreground" />
                            )}
                          </div>
                        )}
                        {isInLibrary(item.id) && (
                          <div className="absolute inset-0 bg-black/50 flex items-center justify-center">
                            <CheckCircle2 className="h-12 w-12 text-green-500" />
                          </div>
                        )}
                        {/* Media type badge */}
                        <Badge
                          className="absolute top-2 right-2"
                          variant="secondary"
                        >
                          {item.title ? (
                            <Film className="h-3 w-3 mr-1" />
                          ) : (
                            <Tv className="h-3 w-3 mr-1" />
                          )}
                          {item.title ? "Movie" : "TV"}
                        </Badge>
                      </div>
                    </Link>

                    <CardContent className="p-3 space-y-2">
                      {/* Title */}
                      <h3 className="font-semibold text-sm line-clamp-2 min-h-[2.5rem]">
                        {item.title || item.name}
                      </h3>

                      {/* Metadata */}
                      <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        {(item.release_date || item.first_air_date) && (
                          <div className="flex items-center gap-1">
                            <Calendar className="h-3 w-3" />
                            <span>
                              {new Date(
                                item.release_date || item.first_air_date || "",
                              ).getFullYear()}
                            </span>
                          </div>
                        )}
                        {item.vote_average > 0 && (
                          <div className="flex items-center gap-1">
                            <Star className="h-3 w-3 fill-yellow-500 text-yellow-500" />
                            <span>{item.vote_average.toFixed(1)}</span>
                          </div>
                        )}
                      </div>

                      {/* Action Button */}
                      {isInLibrary(item.id) ? (
                        <Button
                          className="w-full"
                          size="sm"
                          variant="secondary"
                          asChild
                        >
                          <Link to={`/media/${getLibraryId(item.id)}`}>
                            <ExternalLink className="mr-2 h-3 w-3" />
                            View
                          </Link>
                        </Button>
                      ) : (
                        <Button
                          className="w-full"
                          size="sm"
                          onClick={() => handleAddClick(item)}
                          disabled={isAdded(item.id) || isAdding(item.id)}
                        >
                          {isAdded(item.id) ? (
                            <>
                              <CheckCircle2 className="mr-2 h-3 w-3" />
                              Added
                            </>
                          ) : isAdding(item.id) ? (
                            <>
                              <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                              Adding...
                            </>
                          ) : (
                            <>
                              <Plus className="mr-2 h-3 w-3" />
                              Add
                            </>
                          )}
                        </Button>
                      )}
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Season Selection Modal */}
      <Dialog open={showSeasonModal} onOpenChange={setShowSeasonModal}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Select Seasons to Add</DialogTitle>
            <DialogDescription>
              Choose which seasons you want to add to your library. Leave all
              unchecked to add without specific seasons.
            </DialogDescription>
          </DialogHeader>

          {tvDetails.isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
            </div>
          ) : (
            <div className="space-y-3">
              {/* Select All */}
              <div className="flex items-center space-x-2 pb-2 border-b">
                <Checkbox
                  id="select-all"
                  checked={
                    selectedSeasons.length ===
                      (tvDetails.data?.seasons || []).length &&
                    (tvDetails.data?.seasons || []).length > 0
                  }
                  onCheckedChange={toggleAllSeasons}
                />
                <label
                  htmlFor="select-all"
                  className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer"
                >
                  Select All
                </label>
              </div>

              {/* Individual Seasons */}
              <div className="max-h-60 overflow-y-auto space-y-2">
                {(tvDetails.data?.seasons || [])
                  .filter((season: any) => season.season_number !== 0)
                  .map((season: any) => (
                    <div
                      key={season.id}
                      className="flex items-center space-x-2 py-1"
                    >
                      <Checkbox
                        id={`season-${season.season_number}`}
                        checked={selectedSeasons.includes(season.season_number)}
                        onCheckedChange={() =>
                          toggleSeason(season.season_number)
                        }
                      />
                      <label
                        htmlFor={`season-${season.season_number}`}
                        className="text-sm leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer flex-1"
                      >
                        {season.name}
                        {season.episode_count > 0 && (
                          <span className="text-muted-foreground ml-2">
                            ({season.episode_count} episodes)
                          </span>
                        )}
                      </label>
                    </div>
                  ))}
              </div>
            </div>
          )}

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowSeasonModal(false);
                setSelectedItem(null);
              }}
              disabled={tvDetails.isLoading}
            >
              Cancel
            </Button>
            <Button
              onClick={handleConfirmSeasons}
              disabled={tvDetails.isLoading}
            >
              Add to Library
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
