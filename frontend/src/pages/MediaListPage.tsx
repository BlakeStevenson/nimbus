import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useMediaList } from "@/lib/api/media";
import { MediaTable } from "@/components/media/MediaTable";
import { MediaGrid } from "@/components/media/MediaGrid";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Loader2, LayoutGrid, Table } from "lucide-react";
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
  const [searchParams] = useSearchParams();
  const [viewMode, setViewMode] = useState<"grid" | "table">("grid");

  // Build filters from URL params and props
  const filters: MediaFilters = {
    kind: defaultKind || searchParams.get("kind") || undefined,
    q: searchParams.get("q") || undefined,
  };

  const { data, isLoading, error } = useMediaList(filters);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">{title || "Media Library"}</h1>
          {description && (
            <p className="text-muted-foreground">{description}</p>
          )}
        </div>
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
            <Table className="h-4 w-4 mr-2" />
            Table
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>
            {data
              ? `${data.total} ${data.total === 1 ? "item" : "items"}`
              : "Loading..."}
          </CardTitle>
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

          {data && viewMode === "grid" && <MediaGrid items={data.items} />}
          {data && viewMode === "table" && <MediaTable items={data.items} />}
        </CardContent>
      </Card>
    </div>
  );
}
