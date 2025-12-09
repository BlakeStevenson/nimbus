import { useNavigate } from "react-router-dom";
import { Card, CardContent } from "@/components/ui/card";
import { MediaKindBadge } from "./MediaKindBadge";
import { Film, Tv, Music, Book } from "lucide-react";
import type { MediaItem } from "@/lib/types";

interface MediaGridProps {
  items: MediaItem[];
}

// Placeholder image component for when no poster is available
function PlaceholderImage({ kind }: { kind: string }) {
  const iconClass = "h-24 w-24 text-muted-foreground";

  let Icon = Film;
  if (kind.startsWith("tv_")) {
    Icon = Tv;
  } else if (kind.startsWith("music_")) {
    Icon = Music;
  } else if (kind === "book") {
    Icon = Book;
  }

  return (
    <div className="aspect-[2/3] w-full bg-muted flex items-center justify-center">
      <Icon className={iconClass} />
    </div>
  );
}

// Get poster URL from metadata
function getPosterUrl(item: MediaItem): string | null {
  if (!item.metadata) return null;

  // Check for various poster URL fields
  if (typeof item.metadata === "object") {
    const metadata = item.metadata as Record<string, unknown>;
    if (metadata.poster_url && typeof metadata.poster_url === "string") {
      return metadata.poster_url;
    }
    if (metadata.still_url && typeof metadata.still_url === "string") {
      return metadata.still_url;
    }
  }

  return null;
}

export function MediaGrid({ items }: MediaGridProps) {
  const navigate = useNavigate();

  if (items.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        No media items found
      </div>
    );
  }

  // Sort episodes by episode number if they have one
  const sortedItems = [...items].sort((a, b) => {
    if (a.kind === "tv_episode" && b.kind === "tv_episode") {
      const aEpisode = (a.metadata as Record<string, unknown>)
        ?.episode as number;
      const bEpisode = (b.metadata as Record<string, unknown>)
        ?.episode as number;
      if (aEpisode !== undefined && bEpisode !== undefined) {
        return aEpisode - bEpisode;
      }
    }
    return 0;
  });

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
      {sortedItems.map((item) => {
        const posterUrl = getPosterUrl(item);
        const episodeNumber =
          item.kind === "tv_episode"
            ? ((item.metadata as Record<string, unknown>)?.episode as number)
            : undefined;

        return (
          <Card
            key={item.id}
            className="cursor-pointer hover:shadow-lg transition-shadow overflow-hidden"
            onClick={() => navigate(`/media/${item.id}`)}
          >
            <CardContent className="p-0">
              {/* Poster/Thumbnail */}
              <div className="relative">
                {posterUrl ? (
                  <img
                    src={posterUrl}
                    alt={item.title}
                    className="aspect-[2/3] w-full object-cover"
                    onError={(e) => {
                      // If image fails to load, show placeholder
                      const target = e.target as HTMLImageElement;
                      target.style.display = "none";
                      target.nextElementSibling?.classList.remove("hidden");
                    }}
                  />
                ) : null}
                <div className={posterUrl ? "hidden" : ""}>
                  <PlaceholderImage kind={item.kind} />
                </div>

                {/* Episode number badge for episodes */}
                {episodeNumber !== undefined ? (
                  <div className="absolute top-2 left-2 bg-primary text-primary-foreground text-sm font-bold px-3 py-1 rounded">
                    E{episodeNumber}
                  </div>
                ) : (
                  /* Kind badge overlay for non-episodes */
                  <div className="absolute top-2 right-2">
                    <MediaKindBadge kind={item.kind} />
                  </div>
                )}

                {/* Year badge if available (only for non-episodes) */}
                {item.year && !episodeNumber && (
                  <div className="absolute bottom-2 left-2 bg-black/70 text-white text-xs px-2 py-1 rounded">
                    {item.year}
                  </div>
                )}
              </div>

              {/* Title */}
              <div className="p-3">
                <h3
                  className="font-medium text-sm line-clamp-2"
                  title={item.title}
                >
                  {item.title}
                </h3>

                {item.metadata &&
                typeof item.metadata === "object" &&
                (item.metadata as Record<string, unknown>).rating ? (
                  <div className="flex items-center gap-1 mt-1 text-xs text-muted-foreground">
                    <span className="text-yellow-500">★</span>
                    <span>
                      {Number(
                        (item.metadata as Record<string, unknown>).rating,
                      ).toFixed(1)}
                    </span>
                  </div>
                ) : null}
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
