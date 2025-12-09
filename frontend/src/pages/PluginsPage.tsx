import {
  usePlugins,
  useEnablePlugin,
  useDisablePlugin,
} from "@/lib/api/plugins";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Loader2, Puzzle, Power, PowerOff, AlertCircle } from "lucide-react";
import { useState } from "react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

export function PluginsPage() {
  const { data: plugins, isLoading, error } = usePlugins();
  const enablePlugin = useEnablePlugin();
  const disablePlugin = useDisablePlugin();
  const [actioningPlugin, setActioningPlugin] = useState<string | null>(null);

  const handleToggle = async (pluginId: string, currentlyEnabled: boolean) => {
    setActioningPlugin(pluginId);
    try {
      if (currentlyEnabled) {
        await disablePlugin.mutateAsync(pluginId);
      } else {
        await enablePlugin.mutateAsync(pluginId);
      }
    } catch (err) {
      console.error("Failed to toggle plugin:", err);
    } finally {
      setActioningPlugin(null);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Plugins</h1>
        <p className="text-muted-foreground">
          Manage and configure system plugins
        </p>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>
            Failed to load plugins. If you just enabled plugins, restart the
            server.
          </AlertDescription>
        </Alert>
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
              No plugins have been installed yet. Install plugins in the plugins
              directory and restart the server to see them here.
            </CardDescription>
          </CardHeader>
        </Card>
      )}

      {plugins && plugins.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {plugins.map((plugin) => {
            const isActioning = actioningPlugin === plugin.id;

            return (
              <Card key={plugin.id}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">{plugin.name}</CardTitle>
                    <Badge variant={plugin.enabled ? "default" : "secondary"}>
                      {plugin.enabled ? "Enabled" : "Disabled"}
                    </Badge>
                  </div>
                  <CardDescription>Version {plugin.version}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  {plugin.description && (
                    <p className="text-sm text-muted-foreground">
                      {plugin.description}
                    </p>
                  )}

                  {plugin.capabilities && plugin.capabilities.length > 0 && (
                    <div>
                      <p className="text-xs font-medium text-muted-foreground mb-2">
                        Capabilities:
                      </p>
                      <div className="flex flex-wrap gap-1.5">
                        {plugin.capabilities.map((cap) => (
                          <Badge
                            key={cap}
                            variant="outline"
                            className="text-xs"
                          >
                            {cap}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}

                  <div className="pt-2">
                    <Button
                      onClick={() => handleToggle(plugin.id, plugin.enabled)}
                      disabled={isActioning}
                      variant={plugin.enabled ? "outline" : "default"}
                      className="w-full"
                      size="sm"
                    >
                      {isActioning ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          {plugin.enabled ? "Disabling..." : "Enabling..."}
                        </>
                      ) : plugin.enabled ? (
                        <>
                          <PowerOff className="mr-2 h-4 w-4" />
                          Disable
                        </>
                      ) : (
                        <>
                          <Power className="mr-2 h-4 w-4" />
                          Enable
                        </>
                      )}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
