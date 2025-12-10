import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut } from "../api-client";

export interface TMDBSearchResult {
  id: number;
  title?: string;
  name?: string;
  original_title?: string;
  original_name?: string;
  overview: string;
  poster_path: string | null;
  backdrop_path: string | null;
  release_date?: string;
  first_air_date?: string;
  vote_average: number;
  vote_count: number;
  media_type?: string;
}

export interface TMDBSearchResponse {
  page: number;
  results: TMDBSearchResult[];
  total_pages: number;
  total_results: number;
}

export interface AddToLibraryPayload {
  tmdb_id: number;
  media_type: "movie" | "tv";
  title: string;
  year?: number;
  seasons?: number[];
}

export async function searchMovies(
  query: string,
  year?: string,
): Promise<TMDBSearchResponse> {
  const params = new URLSearchParams({ query });
  if (year) params.append("year", year);
  return apiGet<TMDBSearchResponse>("/api/plugins/tmdb/search/movie", params);
}

export async function searchTV(
  query: string,
  year?: string,
): Promise<TMDBSearchResponse> {
  const params = new URLSearchParams({ query });
  if (year) params.append("year", year);
  return apiGet<TMDBSearchResponse>("/api/plugins/tmdb/search/tv", params);
}

export async function getMovieDetails(id: number): Promise<any> {
  return apiGet(`/api/plugins/tmdb/movie/${id}`);
}

export async function getTVDetails(id: number): Promise<any> {
  return apiGet(`/api/plugins/tmdb/tv/${id}`);
}

export async function getTVSeasonDetails(
  tvId: number,
  seasonNumber: number,
): Promise<any> {
  // Note: TMDB plugin doesn't have a season endpoint yet, so we construct the URL
  return apiGet(`/api/plugins/tmdb/tv/${tvId}/season/${seasonNumber}`);
}

export async function addToLibrary(
  payload: AddToLibraryPayload,
): Promise<{ id: number }> {
  const kind = payload.media_type === "movie" ? "movie" : "tv_series";

  // Prepare metadata
  const metadata: Record<string, any> = {
    tmdb_id: payload.tmdb_id.toString(),
    type: payload.media_type,
  };

  // Create the media item (TV series or movie)
  const mediaItem = await apiPost<{ id: number }>("/api/media", {
    title: payload.title,
    kind,
    year: payload.year,
    metadata,
  });

  // Fetch enriched metadata from TMDB plugin
  const enrichResponse = await apiPost<{
    metadata: Record<string, any>;
    external_ids?: Record<string, any>;
  }>(`/api/plugins/tmdb/enrich/${mediaItem.id}`, {
    tmdb_id: payload.tmdb_id.toString(),
    type: payload.media_type,
  });

  // Update the media item with the enriched metadata and external IDs
  if (enrichResponse.metadata) {
    const updatePayload: Record<string, any> = {
      metadata: enrichResponse.metadata,
    };

    // Add external_ids if provided
    if (enrichResponse.external_ids) {
      updatePayload.external_ids = enrichResponse.external_ids;
    }

    await apiPut(`/api/media/${mediaItem.id}`, updatePayload);
  }

  // If TV series with selected seasons, create season records
  if (
    payload.media_type === "tv" &&
    payload.seasons &&
    payload.seasons.length > 0
  ) {
    // Fetch TV details to get season information
    const tvDetails = await apiGet<any>(
      `/api/plugins/tmdb/tv/${payload.tmdb_id}`,
    );
    const allSeasons = tvDetails.seasons || [];

    // Create season records for selected seasons
    for (const seasonNumber of payload.seasons) {
      const seasonInfo = allSeasons.find(
        (s: any) => s.season_number === seasonNumber,
      );
      if (seasonInfo) {
        // Create season media item
        const seasonMetadata: Record<string, any> = {
          tmdb_id: seasonInfo.id.toString(),
          type: "tv_season",
          season_number: seasonNumber,
          episode_count: seasonInfo.episode_count,
        };

        if (seasonInfo.overview) {
          seasonMetadata.description = seasonInfo.overview;
        }
        if (seasonInfo.poster_path) {
          seasonMetadata.poster_url = `https://image.tmdb.org/t/p/original${seasonInfo.poster_path}`;
        }
        if (seasonInfo.air_date) {
          seasonMetadata.air_date = seasonInfo.air_date;
        }

        const seasonItem = await apiPost<{ id: number }>("/api/media", {
          kind: "tv_season",
          title: seasonInfo.name || `Season ${seasonNumber}`,
          parent_id: mediaItem.id,
          metadata: seasonMetadata,
          external_ids: {
            tmdb: seasonInfo.id.toString(),
          },
        });

        // Fetch season details to get episodes
        const seasonDetails = await apiGet<any>(
          `/api/plugins/tmdb/tv/${payload.tmdb_id}/season/${seasonNumber}`,
        );

        // Create episode records
        if (seasonDetails.episodes && seasonDetails.episodes.length > 0) {
          for (const episode of seasonDetails.episodes) {
            const episodeMetadata: Record<string, any> = {
              tmdb_id: episode.id.toString(),
              type: "tv_episode",
              season: seasonNumber,
              episode: episode.episode_number,
              episode_number: episode.episode_number,
            };

            if (episode.overview) {
              episodeMetadata.description = episode.overview;
            }
            if (episode.still_path) {
              episodeMetadata.still_url = `https://image.tmdb.org/t/p/original${episode.still_path}`;
            }
            if (episode.air_date) {
              episodeMetadata.air_date = episode.air_date;
            }
            if (episode.runtime) {
              episodeMetadata.runtime = episode.runtime;
            }
            if (episode.vote_average) {
              episodeMetadata.vote_average = episode.vote_average;
            }

            await apiPost("/api/media", {
              kind: "tv_episode",
              title: episode.name || `Episode ${episode.episode_number}`,
              parent_id: seasonItem.id,
              metadata: episodeMetadata,
              external_ids: enrichResponse.external_ids || {
                tmdb: episode.id.toString(),
              },
            });
          }
        }
      }
    }
  }

  return mediaItem;
}

export function useSearchMovies(query: string, year?: string) {
  return useQuery({
    queryKey: ["tmdb", "search", "movie", query, year],
    queryFn: () => searchMovies(query, year),
    enabled: query.length > 0,
    staleTime: 5 * 60 * 1000,
  });
}

export function useSearchTV(query: string, year?: string) {
  return useQuery({
    queryKey: ["tmdb", "search", "tv", query, year],
    queryFn: () => searchTV(query, year),
    enabled: query.length > 0,
    staleTime: 5 * 60 * 1000,
  });
}

export function useMovieDetails(id: number | null) {
  return useQuery({
    queryKey: ["tmdb", "movie", id],
    queryFn: () => getMovieDetails(id!),
    enabled: id !== null,
    staleTime: 10 * 60 * 1000,
  });
}

export function useTVDetails(id: number | null) {
  return useQuery({
    queryKey: ["tmdb", "tv", id],
    queryFn: () => getTVDetails(id!),
    enabled: id !== null,
    staleTime: 10 * 60 * 1000,
  });
}

export function useTVSeasonDetails(
  tvId: number | null,
  seasonNumber: number | null,
) {
  return useQuery({
    queryKey: ["tmdb", "season", tvId, seasonNumber],
    queryFn: () => getTVSeasonDetails(tvId!, seasonNumber!),
    enabled: tvId !== null && seasonNumber !== null,
    staleTime: 10 * 60 * 1000,
  });
}

export function useAddToLibrary() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: addToLibrary,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["media"] });
    },
  });
}
