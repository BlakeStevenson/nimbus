import { usePlugins } from '@/lib/api/plugins';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Loader2, Puzzle } from 'lucide-react';

export function PluginsPage() {
  const { data: plugins, isLoading, error } = usePlugins();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Plugins</h1>
        <p className="text-muted-foreground">
          Manage and configure system plugins
        </p>
      </div>

      {error && (
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-destructive">
              Failed to load plugins
            </p>
          </CardContent>
        </Card>
      )}

      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {plugins && plugins.length === 0 && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Puzzle className="h-5 w-5 text-muted-foreground" />
              <CardTitle>No Plugins Installed</CardTitle>
            </div>
            <CardDescription>
              Plugin support is coming soon. This page will allow you to manage
              plugin-based extensions to Nimbus.
            </CardDescription>
          </CardHeader>
        </Card>
      )}

      {plugins && plugins.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2">
          {plugins.map((plugin) => (
            <Card key={plugin.id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>{plugin.name}</CardTitle>
                  <Badge variant={plugin.enabled ? 'default' : 'secondary'}>
                    {plugin.enabled ? 'Enabled' : 'Disabled'}
                  </Badge>
                </div>
                <CardDescription>
                  Version {plugin.version}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                {plugin.description && (
                  <p className="text-sm">{plugin.description}</p>
                )}
                {plugin.capabilities && plugin.capabilities.length > 0 && (
                  <div>
                    <p className="text-sm font-medium mb-2">Capabilities:</p>
                    <div className="flex flex-wrap gap-2">
                      {plugin.capabilities.map((cap) => (
                        <Badge key={cap} variant="outline">
                          {cap}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
