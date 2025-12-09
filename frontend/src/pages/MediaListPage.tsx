import { useState, useEffect, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { useMediaList } from "@/lib/api/media";
import { MediaTable } from "@/components/media/MediaTable";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Loader2 } from "lucide-react";
import type { MediaFilters } from "@/lib/types";

interface MediaListPageProps {
  defaultKind?: string;
  title?: string;
  description?: string;
}

export function MediaListPage({
  defaultKind,
  title,
  description,
}: MediaListPageProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const [searchInput, setSearchInput] = useState(searchParams.get("q") || "");
  const debounceTimerRef = useRef<NodeJS.Timeout>();

  // Build filters from URL params and props
  const filters: MediaFilters = {
    kind: defaultKind || searchParams.get("kind") || undefined,
    q: searchParams.get("q") || undefined,
  };

  const { data, isLoading, error } = useMediaList(filters);

  // Sync search input with URL when URL changes externally
  useEffect(() => {
    const qFromUrl = searchParams.get("q") || "";
    if (qFromUrl !== searchInput) {
      setSearchInput(qFromUrl);
    }
  }, [searchParams]);

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  const handleKindChange = (value: string) => {
    const newParams = new URLSearchParams(searchParams);
    if (value === "all") {
      newParams.delete("kind");
    } else {
      newParams.set("kind", value);
    }
    setSearchParams(newParams);
  };

  const handleSearchChange = (value: string) => {
    setSearchInput(value);

    // Clear existing timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // Set new timer to update URL after 500ms
    debounceTimerRef.current = setTimeout(() => {
      const newParams = new URLSearchParams(searchParams);
      if (value) {
        newParams.set("q", value);
      } else {
        newParams.delete("q");
      }
      setSearchParams(newParams);
    }, 500);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">{title || "Media Library"}</h1>
        {description && <p className="text-muted-foreground">{description}</p>}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Filters</CardTitle>
          <CardDescription>Filter and search your media</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="kind">Media Type</Label>
              <Select
                value={filters.kind || "all"}
                onValueChange={handleKindChange}
                disabled={!!defaultKind}
              >
                <SelectTrigger id="kind">
                  <SelectValue placeholder="Select type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Types</SelectItem>
                  <SelectItem value="movie">Movies</SelectItem>
                  <SelectItem value="tv_series">TV Series</SelectItem>
                  <SelectItem value="tv_season">TV Seasons</SelectItem>
                  <SelectItem value="tv_episode">TV Episodes</SelectItem>
                  <SelectItem value="music_artist">Artists</SelectItem>
                  <SelectItem value="music_album">Albums</SelectItem>
                  <SelectItem value="music_track">Tracks</SelectItem>
                  <SelectItem value="book">Books</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="search">Search</Label>
              <Input
                id="search"
                type="search"
                placeholder="Search by title..."
                value={searchInput}
                onChange={(e) => handleSearchChange(e.target.value)}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Results</CardTitle>
          <CardDescription>
            {data
              ? `${data.total} ${data.total === 1 ? "item" : "items"} found`
              : "Loading..."}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {error && (
            <p className="text-sm text-destructive">
              Failed to load media items
            </p>
          )}

          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          )}

          {data && <MediaTable items={data.items} />}
        </CardContent>
      </Card>
    </div>
  );
}
