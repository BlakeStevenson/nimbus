import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useMovieDetails, useTVDetails, useAddToLibrary } from "@/lib/api/tmdb";
import { useTMDBInLibrary } from "@/lib/api/media";
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
import {
  Loader2,
  ArrowLeft,
  Plus,
  ExternalLink,
  Star,
  Calendar,
  Clock,
  Film,
  Tv,
} from "lucide-react";
import { cn } from "@/lib/utils";

const TMDB_IMAGE_BASE_URL = "https://image.tmdb.org/t/p/original";

export default function TMDBDetailPage() {
  const { type, id } = useParams<{ type: string; id: string }>();
  const navigate = useNavigate();
  const tmdbId = parseInt(id || "0", 10);
  const isMovie = type === "movie";

  const [isAdding, setIsAdding] = useState(false);
  const [showSeasonModal, setShowSeasonModal] = useState(false);
  const [selectedSeasons, setSelectedSeasons] = useState<number[]>([]);

  const movieDetails = useMovieDetails(isMovie ? tmdbId : null);
  const tvDetails = useTVDetails(!isMovie ? tmdbId : null);
  const { data: libraryCheck } = useTMDBInLibrary([tmdbId]);
  const addToLibrary = useAddToLibrary();

  const details = isMovie ? movieDetails.data : tvDetails.data;
  const isLoading = isMovie ? movieDetails.isLoading : tvDetails.isLoading;
  const error = isMovie ? movieDetails.error : tvDetails.error;

  const libraryId = libraryCheck?.[tmdbId];
  const isInLibrary = !!libraryId;

  const seasons = !isMovie && details ? (details as any).seasons || [] : [];

  const handleAddToLibrary = async (seasonsToAdd?: number[]) => {
    if (!details) return;

    setIsAdding(true);
    try {
      const title = isMovie ? (details as any).title : (details as any).name;
      const year = isMovie
        ? new Date((details as any).release_date || "").getFullYear()
        : new Date((details as any).first_air_date || "").getFullYear();

      const result = await addToLibrary.mutateAsync({
        tmdb_id: tmdbId,
        media_type: isMovie ? "movie" : "tv",
        title,
        year: year || undefined,
        seasons: seasonsToAdd,
      });

      // Navigate to the newly created media item
      navigate(`/media/${result.id}`);
    } catch (error) {
      console.error("Failed to add to library:", error);
      setIsAdding(false);
    }
  };

  const handleAddClick = () => {
    if (!isMovie && seasons.length > 0) {
      // Show season selection modal for TV shows
      setSelectedSeasons([]);
      setShowSeasonModal(true);
    } else {
      // Add movie directly
      handleAddToLibrary();
    }
  };

  const handleConfirmSeasons = () => {
    setShowSeasonModal(false);
    handleAddToLibrary(
      selectedSeasons.length > 0 ? selectedSeasons : undefined,
    );
  };

  const toggleSeason = (seasonNumber: number) => {
    setSelectedSeasons((prev) =>
      prev.includes(seasonNumber)
        ? prev.filter((s) => s !== seasonNumber)
        : [...prev, seasonNumber],
    );
  };

  const toggleAllSeasons = () => {
    if (selectedSeasons.length === seasons.length) {
      setSelectedSeasons([]);
    } else {
      setSelectedSeasons(seasons.map((s: any) => s.season_number));
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (error || !details) {
    return (
      <div className="container mx-auto py-16 text-center">
        <p className="text-destructive">Failed to load details</p>
        <Button onClick={() => navigate(-1)} className="mt-4">
          <ArrowLeft className="mr-2 h-4 w-4" />
          Go Back
        </Button>
      </div>
    );
  }

  const title = isMovie ? (details as any).title : (details as any).name;
  const releaseDate = isMovie
    ? (details as any).release_date
    : (details as any).first_air_date;
  const backdropPath = (details as any).backdrop_path;
  const posterPath = (details as any).poster_path;
  const overview = (details as any).overview;
  const voteAverage = (details as any).vote_average;
  const genres = (details as any).genres || [];
  const runtime = isMovie ? (details as any).runtime : null;
  const numberOfSeasons = !isMovie ? (details as any).number_of_seasons : null;
  const numberOfEpisodes = !isMovie
    ? (details as any).number_of_episodes
    : null;

  return (
    <div className="min-h-screen">
      {/* Backdrop */}
      {backdropPath ? (
        <div className="relative h-64 w-full">
          <img
            src={`${TMDB_IMAGE_BASE_URL}${backdropPath}`}
            alt={title}
            className="w-full h-full object-cover"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-background via-background/60 to-transparent" />
        </div>
      ) : (
        <div className="h-24 bg-gradient-to-b from-muted to-background" />
      )}

      <div className="container mx-auto px-4 py-6">
        {/* Back Button */}
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate(-1)}
          className="mb-4"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>

        {/* Title and Action Button */}
        <div className="flex items-start justify-between gap-4 mb-6">
          <div>
            <h1 className="text-4xl font-bold mb-2">{title}</h1>
            <div className="flex items-center gap-3 text-muted-foreground">
              {releaseDate && (
                <div className="flex items-center gap-1">
                  <Calendar className="h-4 w-4" />
                  <span>{new Date(releaseDate).getFullYear()}</span>
                </div>
              )}
              {runtime && (
                <div className="flex items-center gap-1">
                  <Clock className="h-4 w-4" />
                  <span>{runtime} min</span>
                </div>
              )}
              {numberOfSeasons && (
                <div className="flex items-center gap-1">
                  <Tv className="h-4 w-4" />
                  <span>
                    {numberOfSeasons}{" "}
                    {numberOfSeasons === 1 ? "Season" : "Seasons"}
                    {numberOfEpisodes && ` · ${numberOfEpisodes} Episodes`}
                  </span>
                </div>
              )}
              {voteAverage > 0 && (
                <div className="flex items-center gap-1">
                  <Star className="h-4 w-4 fill-yellow-500 text-yellow-500" />
                  <span className="font-semibold">
                    {voteAverage.toFixed(1)}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Action Button */}
          <div className="flex-shrink-0">
            {isInLibrary ? (
              <Button size="lg" variant="secondary" asChild>
                <Link to={`/media/${libraryId}`}>
                  <ExternalLink className="mr-2 h-5 w-5" />
                  View in Library
                </Link>
              </Button>
            ) : (
              <Button size="lg" onClick={handleAddClick} disabled={isAdding}>
                {isAdding ? (
                  <>
                    <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                    Adding to Library...
                  </>
                ) : (
                  <>
                    <Plus className="mr-2 h-5 w-5" />
                    Add to Library
                  </>
                )}
              </Button>
            )}
          </div>
        </div>

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

            <div className="space-y-3">
              {/* Select All */}
              <div className="flex items-center space-x-2 pb-2 border-b">
                <Checkbox
                  id="select-all"
                  checked={
                    selectedSeasons.length === seasons.length &&
                    seasons.length > 0
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
                {seasons
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

            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setShowSeasonModal(false)}
              >
                Cancel
              </Button>
              <Button onClick={handleConfirmSeasons} disabled={isAdding}>
                {isAdding ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Adding...
                  </>
                ) : (
                  <>Add to Library</>
                )}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Genres */}
        {genres.length > 0 && (
          <div className="flex flex-wrap gap-2 mb-6">
            {genres.map((genre: any) => (
              <Badge key={genre.id} variant="secondary">
                {genre.name}
              </Badge>
            ))}
          </div>
        )}

        {/* Main Content: Poster and Details Side by Side */}
        <div className="flex gap-6">
          {/* Poster */}
          <Card className="w-48 flex-shrink-0 overflow-hidden self-start">
            {posterPath ? (
              <img
                src={`${TMDB_IMAGE_BASE_URL}${posterPath}`}
                alt={title}
                className="w-full max-h-80 object-cover"
              />
            ) : (
              <div className="aspect-[2/3] bg-muted flex items-center justify-center">
                {isMovie ? (
                  <Film className="h-16 w-16 text-muted-foreground" />
                ) : (
                  <Tv className="h-16 w-16 text-muted-foreground" />
                )}
              </div>
            )}
          </Card>

          {/* Details Column */}
          <div className="flex-1 space-y-4">
            {/* Overview */}
            {overview && (
              <Card>
                <CardHeader>
                  <CardTitle>Overview</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-muted-foreground leading-relaxed">
                    {overview}
                  </p>
                </CardContent>
              </Card>
            )}

            {/* Cast */}
            {(details as any).credits?.cast &&
              (details as any).credits.cast.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle>Cast</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
                      {(details as any).credits.cast
                        .slice(0, 8)
                        .map((person: any) => (
                          <div
                            key={person.id}
                            className="flex items-center gap-3"
                          >
                            {person.profile_path ? (
                              <img
                                src={`https://image.tmdb.org/t/p/w185${person.profile_path}`}
                                alt={person.name}
                                className="w-12 h-12 rounded-full object-cover"
                              />
                            ) : (
                              <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center">
                                <span className="text-xs font-semibold">
                                  {person.name.substring(0, 2)}
                                </span>
                              </div>
                            )}
                            <div className="min-w-0">
                              <p className="font-medium text-sm truncate">
                                {person.name}
                              </p>
                              <p className="text-xs text-muted-foreground truncate">
                                {person.character}
                              </p>
                            </div>
                          </div>
                        ))}
                    </div>
                  </CardContent>
                </Card>
              )}
          </div>
        </div>
      </div>
    </div>
  );
}
