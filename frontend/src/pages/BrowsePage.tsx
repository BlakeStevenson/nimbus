import { useState, useMemo, useEffect } from "react";
import { useSearchMovies, useSearchTV, useAddToLibrary } from "@/lib/api/tmdb";
import type { TMDBSearchResult } from "@/lib/api/tmdb";
import { useTMDBInLibrary } from "@/lib/api/media";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";

import { ScrollArea } from "@/components/ui/scroll-area";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Search,
  Film,
  Tv,
  Plus,
  Loader2,
  Star,
  Calendar,
  CheckCircle2,
  ExternalLink,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Link, useSearchParams } from "react-router-dom";

type MediaType = "movie" | "tv";

const TMDB_IMAGE_BASE_URL = "https://image.tmdb.org/t/p/w500";

export default function BrowsePage() {
  const [searchParams, setSearchParams] = useSearchParams();

  // Initialize state from URL params
  const initialQuery = searchParams.get("q") || "";
  const initialType = (searchParams.get("type") as MediaType) || "movie";

  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const [debouncedQuery, setDebouncedQuery] = useState(initialQuery);
  const [mediaType, setMediaType] = useState<MediaType>(
    initialType === "tv" ? "tv" : "movie",
  );
  const [addedItems, setAddedItems] = useState<Set<number>>(new Set());
  const [addingItem, setAddingItem] = useState<number | null>(null);

  const addToLibrary = useAddToLibrary();

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery);
    }, 500);

    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Update URL params when state changes
  useEffect(() => {
    const params = new URLSearchParams();
    if (debouncedQuery) {
      params.set("q", debouncedQuery);
    }
    params.set("type", mediaType);
    setSearchParams(params, { replace: true });
  }, [debouncedQuery, mediaType, setSearchParams]);

  // Fetch search results based on media type
  const movieResults = useSearchMovies(
    mediaType === "movie" ? debouncedQuery : "",
  );
  const tvResults = useSearchTV(mediaType === "tv" ? debouncedQuery : "");

  const searchResults = mediaType === "movie" ? movieResults : tvResults;

  // Extract TMDB IDs from search results
  const tmdbIds = useMemo(() => {
    return searchResults.data?.results.map((item) => item.id) || [];
  }, [searchResults.data]);

  // Check which items are already in the library
  const { data: libraryCheck } = useTMDBInLibrary(tmdbIds);

  const handleAddToLibrary = async (item: TMDBSearchResult) => {
    setAddingItem(item.id);
    try {
      const title = item.title || item.name || "";
      const year = item.release_date
        ? new Date(item.release_date).getFullYear()
        : item.first_air_date
          ? new Date(item.first_air_date).getFullYear()
          : undefined;

      await addToLibrary.mutateAsync({
        tmdb_id: item.id,
        media_type: mediaType,
        title,
        year,
      });

      // Mark as added
      setAddedItems((prev) => new Set(prev).add(item.id));
    } catch (error) {
      console.error("Failed to add to library:", error);
    } finally {
      setAddingItem(null);
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

  return (
    <div className="container mx-auto py-6 space-y-4">
      {/* Compact Header with Search */}
      <div className="flex items-center gap-4">
        <div className="flex-1">
          <h1 className="text-2xl font-bold mb-2">Browse</h1>
          <div className="flex gap-2 items-center">
            {/* Media Type Toggle */}
            <Button
              variant={mediaType === "movie" ? "default" : "outline"}
              onClick={() => setMediaType("movie")}
              size="sm"
            >
              <Film className="mr-2 h-4 w-4" />
              Movies
            </Button>
            <Button
              variant={mediaType === "tv" ? "default" : "outline"}
              onClick={() => setMediaType("tv")}
              size="sm"
            >
              <Tv className="mr-2 h-4 w-4" />
              TV Shows
            </Button>

            {/* Search Input */}
            <div className="relative flex-1 max-w-md">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                type="text"
                placeholder={`Search ${mediaType === "movie" ? "movies" : "TV shows"}...`}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10 h-9"
              />
            </div>
          </div>
        </div>
      </div>

      {/* Search Results */}
      {searchResults.isLoading && (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      )}

      {searchResults.isError && (
        <Alert variant="destructive">
          <AlertDescription>
            Failed to search TMDB. Check plugin configuration.
          </AlertDescription>
        </Alert>
      )}

      {searchResults.data &&
        searchResults.data.results.length === 0 &&
        debouncedQuery && (
          <div className="text-center py-8 text-muted-foreground">
            No results found for "{debouncedQuery}"
          </div>
        )}

      {searchResults.data && searchResults.data.results.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-semibold">
              {searchResults.data.total_results} Results
            </h2>
          </div>

          <ScrollArea className="h-[calc(100vh-220px)]">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 pr-4">
              {searchResults.data.results.map((item) => (
                <Card
                  key={item.id}
                  className={cn(
                    "overflow-hidden hover:shadow-lg transition-shadow",
                    (isAdded(item.id) || isInLibrary(item.id)) && "opacity-60",
                  )}
                >
                  {/* Poster Image */}
                  <div className="relative aspect-[2/3] bg-muted">
                    {item.poster_path ? (
                      <img
                        src={`${TMDB_IMAGE_BASE_URL}${item.poster_path}`}
                        alt={item.title || item.name}
                        className="w-full h-full object-cover"
                      />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center">
                        {mediaType === "movie" ? (
                          <Film className="h-16 w-16 text-muted-foreground" />
                        ) : (
                          <Tv className="h-16 w-16 text-muted-foreground" />
                        )}
                      </div>
                    )}
                    {isAdded(item.id) && (
                      <div className="absolute inset-0 bg-black/50 flex items-center justify-center">
                        <CheckCircle2 className="h-12 w-12 text-green-500" />
                      </div>
                    )}
                  </div>

                  <CardContent className="p-4 space-y-2">
                    {/* Title */}
                    <h3 className="font-semibold line-clamp-2 min-h-[2.5rem]">
                      {item.title || item.name}
                    </h3>

                    {/* Metadata */}
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
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

                    {/* Overview */}
                    {item.overview && (
                      <p className="text-xs text-muted-foreground line-clamp-3">
                        {item.overview}
                      </p>
                    )}

                    {/* Add Button or View Link */}
                    {isInLibrary(item.id) ? (
                      <Button
                        className="w-full mt-2"
                        size="sm"
                        variant="secondary"
                        asChild
                      >
                        <Link to={`/media/${getLibraryId(item.id)}`}>
                          <ExternalLink className="mr-2 h-4 w-4" />
                          View in Library
                        </Link>
                      </Button>
                    ) : (
                      <Button
                        className="w-full mt-2"
                        size="sm"
                        onClick={() => handleAddToLibrary(item)}
                        disabled={isAdded(item.id) || isAdding(item.id)}
                      >
                        {isAdded(item.id) ? (
                          <>
                            <CheckCircle2 className="mr-2 h-4 w-4" />
                            Added
                          </>
                        ) : isAdding(item.id) ? (
                          <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Adding...
                          </>
                        ) : (
                          <>
                            <Plus className="mr-2 h-4 w-4" />
                            Add to Library
                          </>
                        )}
                      </Button>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          </ScrollArea>
        </div>
      )}

      {/* Initial State */}
      {!debouncedQuery && (
        <div className="text-center py-16 text-muted-foreground">
          <Search className="h-16 w-16 mx-auto mb-4 opacity-20" />
          <p className="text-sm">
            Search for {mediaType === "movie" ? "movies" : "TV shows"} to add to
            your library
          </p>
        </div>
      )}
    </div>
  );
}
