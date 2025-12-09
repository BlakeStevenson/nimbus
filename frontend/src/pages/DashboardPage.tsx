import { useMediaStats } from '@/lib/api/media';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Loader2, Film, Tv, Music, BookOpen, Disc, Radio } from 'lucide-react';
import { formatMediaKind } from '@/lib/utils';

const kindIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  movie: Film,
  tv_series: Tv,
  tv_episode: Tv,
  music_album: Disc,
  music_track: Music,
  book: BookOpen,
};

export function DashboardPage() {
  const { data: stats, isLoading, error } = useMediaStats();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">
          Overview of your media library
        </p>
      </div>

      {error && (
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-destructive">
              Failed to load statistics
            </p>
          </CardContent>
        </Card>
      )}

      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {stats && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {stats.map((stat) => {
            const Icon = kindIcons[stat.kind] || Radio;
            return (
              <Card key={stat.kind}>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    {formatMediaKind(stat.kind)}
                  </CardTitle>
                  <Icon className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{stat.count.toLocaleString()}</div>
                  <p className="text-xs text-muted-foreground">
                    {stat.count === 1 ? 'item' : 'items'} in library
                  </p>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Welcome to Nimbus</CardTitle>
          <CardDescription>
            Your self-hosted media management system
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          <p className="text-sm text-muted-foreground">
            Use the sidebar to navigate through your media library, configure settings,
            and manage plugins.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
