import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatDate(dateString: string | null | undefined): string {
  if (!dateString) {
    return "—";
  }

  const date = new Date(dateString);

  // Check if date is valid
  if (isNaN(date.getTime())) {
    return "—";
  }

  return new Intl.DateTimeFormat("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function formatMediaKind(kind: string): string {
  const kindMap: Record<string, string> = {
    movie: "Movie",
    tv_series: "TV Series",
    tv_season: "TV Season",
    tv_episode: "TV Episode",
    music_artist: "Artist",
    music_album: "Album",
    music_track: "Track",
    book: "Book",
  };
  return kindMap[kind] || kind;
}

export function getMediaKindColor(kind: string): string {
  const colorMap: Record<string, string> = {
    movie: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
    tv_series:
      "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
    tv_season:
      "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
    tv_episode:
      "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
    music_artist:
      "bg-pink-100 text-pink-800 dark:bg-pink-900 dark:text-pink-300",
    music_album:
      "bg-pink-100 text-pink-800 dark:bg-pink-900 dark:text-pink-300",
    music_track:
      "bg-pink-100 text-pink-800 dark:bg-pink-900 dark:text-pink-300",
    book: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  };
  return (
    colorMap[kind] ||
    "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300"
  );
}
