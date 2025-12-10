import { useState, useEffect } from "react";
import { useAllConfig, useUpdateMultipleConfigs } from "@/lib/api/config";
import { usePluginsWithConfig } from "@/lib/api/plugins";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Loader2, Save } from "lucide-react";
import type { ConfigValue } from "@/lib/types";
import type { PluginConfigField } from "@/lib/api/plugins";
import { ObjectListField } from "@/components/config/ObjectListField";
import { pluginSchemas } from "@/config/plugin-schemas";

export function ConfigurationPage() {
  const { data: configs, isLoading: configsLoading, error } = useAllConfig();
  const { data: pluginsWithConfig, isLoading: pluginsLoading } =
    usePluginsWithConfig();
  const updateConfigs = useUpdateMultipleConfigs();
  const [editedValues, setEditedValues] = useState<Record<string, any>>({});
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    if (configs) {
      const initialValues: Record<string, any> = {};
      configs.forEach((config) => {
        initialValues[config.key] = config.value;
      });
      setEditedValues(initialValues);
    }
  }, [configs]);

  const handleValueChange = (key: string, value: any) => {
    setEditedValues((prev) => ({
      ...prev,
      [key]: value,
    }));
    setHasChanges(true);
  };

  const handleSave = async () => {
    try {
      await updateConfigs.mutateAsync(editedValues);
      setHasChanges(false);
    } catch (error) {
      console.error("Failed to save configs:", error);
    }
  };

  const handleReset = () => {
    if (configs) {
      const initialValues: Record<string, any> = {};
      configs.forEach((config) => {
        initialValues[config.key] = config.value;
      });
      setEditedValues(initialValues);
      setHasChanges(false);
    }
  };

  const renderConfigInput = (config: ConfigValue) => {
    const type = config.metadata?.type || "text";
    const currentValue = editedValues[config.key] ?? config.value;

    switch (type) {
      case "boolean":
        return (
          <div className="flex items-center space-x-2">
            <Switch
              id={config.key}
              checked={currentValue === true}
              onCheckedChange={(checked: boolean) =>
                handleValueChange(config.key, checked)
              }
            />
            <Label htmlFor={config.key} className="cursor-pointer">
              {currentValue ? "Enabled" : "Disabled"}
            </Label>
          </div>
        );

      case "number":
        return (
          <Input
            type="number"
            value={currentValue ?? ""}
            onChange={(e) =>
              handleValueChange(
                config.key,
                e.target.value ? Number(e.target.value) : null,
              )
            }
          />
        );

      case "select":
        if (!config.metadata?.values || config.metadata.values.length === 0) {
          return (
            <Input
              type="text"
              value={currentValue ?? ""}
              onChange={(e) => handleValueChange(config.key, e.target.value)}
            />
          );
        }
        return (
          <Select
            value={currentValue ?? ""}
            onValueChange={(value) => handleValueChange(config.key, value)}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select a value" />
            </SelectTrigger>
            <SelectContent>
              {config.metadata.values.map((option) => (
                <SelectItem key={option} value={option}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        );

      case "multi":
        if (!config.metadata?.values || config.metadata.values.length === 0) {
          return (
            <Input
              type="text"
              value={JSON.stringify(currentValue ?? [])}
              onChange={(e) => {
                try {
                  const parsed = JSON.parse(e.target.value);
                  handleValueChange(config.key, parsed);
                } catch {
                  // Ignore invalid JSON
                }
              }}
            />
          );
        }
        const selectedValues = Array.isArray(currentValue) ? currentValue : [];
        return (
          <div className="space-y-2">
            {config.metadata.values.map((option) => (
              <div key={option} className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id={`${config.key}-${option}`}
                  checked={selectedValues.includes(option)}
                  onChange={(e) => {
                    const newValues = e.target.checked
                      ? [...selectedValues, option]
                      : selectedValues.filter((v) => v !== option);
                    handleValueChange(config.key, newValues);
                  }}
                  className="h-4 w-4 rounded border-gray-300"
                />
                <label
                  htmlFor={`${config.key}-${option}`}
                  className="text-sm cursor-pointer"
                >
                  {option}
                </label>
              </div>
            ))}
          </div>
        );

      case "text":
      default:
        return (
          <Input
            type="text"
            value={currentValue ?? ""}
            onChange={(e) => handleValueChange(config.key, e.target.value)}
          />
        );
    }
  };

  const renderPluginConfigField = (
    field: PluginConfigField,
    pluginId: string,
  ) => {
    const key = field.key;
    const currentValue = editedValues[key] ?? field.defaultValue ?? "";

    // Use generic ObjectListField for custom type with schema
    if (field.type === "custom") {
      const schemaKey = `${pluginId}:${key}`;
      const schemaConfig = pluginSchemas[schemaKey];

      if (schemaConfig) {
        return (
          <ObjectListField
            value={currentValue}
            onChange={(value) => handleValueChange(key, value)}
            {...schemaConfig}
          />
        );
      }
    }

    switch (field.type) {
      case "boolean":
        return (
          <div className="flex items-center space-x-2">
            <Switch
              id={key}
              checked={currentValue === true || currentValue === "true"}
              onCheckedChange={(checked: boolean) =>
                handleValueChange(key, checked)
              }
            />
            <Label htmlFor={key} className="cursor-pointer">
              {currentValue ? "Enabled" : "Disabled"}
            </Label>
          </div>
        );

      case "number":
        return (
          <Input
            type="number"
            value={currentValue ?? ""}
            onChange={(e) =>
              handleValueChange(
                key,
                e.target.value ? Number(e.target.value) : null,
              )
            }
            placeholder={field.placeholder}
            min={field.validation?.min}
            max={field.validation?.max}
          />
        );

      case "password":
        return (
          <Input
            type="password"
            value={currentValue ?? ""}
            onChange={(e) => handleValueChange(key, e.target.value)}
            placeholder={field.placeholder}
          />
        );

      case "textarea":
        return (
          <Textarea
            value={currentValue ?? ""}
            onChange={(e) => handleValueChange(key, e.target.value)}
            placeholder={field.placeholder}
            rows={4}
          />
        );

      case "select":
        if (!field.options || field.options.length === 0) {
          return (
            <Input
              type="text"
              value={currentValue ?? ""}
              onChange={(e) => handleValueChange(key, e.target.value)}
              placeholder={field.placeholder}
            />
          );
        }
        return (
          <Select
            value={currentValue ?? ""}
            onValueChange={(value) => handleValueChange(key, value)}
          >
            <SelectTrigger>
              <SelectValue
                placeholder={field.placeholder || "Select a value"}
              />
            </SelectTrigger>
            <SelectContent>
              {field.options.map((option) => (
                <SelectItem key={option} value={option}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        );

      case "array":
        const arrayValue = Array.isArray(currentValue) ? currentValue : [];
        return (
          <Textarea
            value={JSON.stringify(arrayValue, null, 2)}
            onChange={(e) => {
              try {
                const parsed = JSON.parse(e.target.value);
                if (Array.isArray(parsed)) {
                  handleValueChange(key, parsed);
                }
              } catch {
                // Ignore invalid JSON
              }
            }}
            placeholder={field.placeholder || "[]"}
            rows={6}
            className="font-mono text-sm"
          />
        );

      case "text":
      default:
        return (
          <Input
            type="text"
            value={currentValue ?? ""}
            onChange={(e) => handleValueChange(key, e.target.value)}
            placeholder={field.placeholder}
            pattern={field.validation?.pattern}
          />
        );
    }
  };

  if (error) {
    return (
      <div className="space-y-6 p-6">
        <div>
          <h1 className="text-3xl font-bold">Configuration</h1>
          <p className="text-muted-foreground">
            Manage system and plugin settings
          </p>
        </div>
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-destructive">
              Failed to load configuration
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const isLoading = configsLoading || pluginsLoading;

  if (isLoading) {
    return (
      <div className="space-y-6 p-6">
        <div>
          <h1 className="text-3xl font-bold">Configuration</h1>
          <p className="text-muted-foreground">
            Manage system and plugin settings
          </p>
        </div>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    );
  }

  // Separate configs by category
  const systemConfigs =
    configs?.filter(
      (c) => !c.key.startsWith("plugins.") && !c.key.startsWith("downloads."),
    ) || [];

  const downloadConfigs =
    configs?.filter((c) => c.key.startsWith("downloads.")) || [];

  const pluginsWithConfigSections =
    pluginsWithConfig?.filter((p) => p.configSection) || [];

  // Group download configs by section
  const downloadSections = downloadConfigs.reduce(
    (acc, config) => {
      const section = config.metadata?.section || "General";
      if (!acc[section]) {
        acc[section] = [];
      }
      acc[section].push(config);
      return acc;
    },
    {} as Record<string, typeof downloadConfigs>,
  );

  // Debug logging
  console.log("pluginsWithConfig:", pluginsWithConfig);
  console.log("pluginsWithConfigSections:", pluginsWithConfigSections);

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Configuration</h1>
          <p className="text-muted-foreground">
            Manage system and plugin settings
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={handleReset}
            disabled={!hasChanges}
          >
            Reset
          </Button>
          <Button
            onClick={handleSave}
            disabled={!hasChanges || updateConfigs.isPending}
          >
            {updateConfigs.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Save className="mr-2 h-4 w-4" />
                Save Changes
              </>
            )}
          </Button>
        </div>
      </div>

      <Tabs defaultValue="general" className="w-full">
        <TabsList>
          <TabsTrigger value="general">General</TabsTrigger>
          {downloadConfigs.length > 0 && (
            <TabsTrigger value="downloads">Downloads</TabsTrigger>
          )}
          {pluginsWithConfigSections.length > 0 && (
            <TabsTrigger value="plugins">Plugins</TabsTrigger>
          )}
        </TabsList>

        <TabsContent value="general" className="space-y-4">
          {systemConfigs.length > 0 ? (
            systemConfigs.map((config) => (
              <Card key={config.key}>
                <CardHeader>
                  <CardTitle>{config.metadata?.title || config.key}</CardTitle>
                  {config.metadata?.description && (
                    <CardDescription>
                      {config.metadata.description}
                    </CardDescription>
                  )}
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {renderConfigInput(config)}
                    <p className="text-xs text-muted-foreground">
                      Key:{" "}
                      <code className="bg-muted px-1 py-0.5 rounded">
                        {config.key}
                      </code>
                    </p>
                  </div>
                </CardContent>
              </Card>
            ))
          ) : (
            <Card>
              <CardContent className="pt-6">
                <p className="text-sm text-muted-foreground">
                  No system configuration settings found
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {downloadConfigs.length > 0 && (
          <TabsContent value="downloads" className="space-y-4">
            {Object.entries(downloadSections).map(
              ([sectionName, sectionConfigs]) => (
                <Card key={sectionName}>
                  <CardHeader>
                    <CardTitle>{sectionName}</CardTitle>
                    <CardDescription>
                      Configure {sectionName.toLowerCase()} settings for
                      downloaded media
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    {sectionConfigs.map((config) => (
                      <div key={config.key} className="space-y-2">
                        <Label htmlFor={config.key}>
                          {config.metadata?.title || config.key}
                        </Label>
                        {config.metadata?.description && (
                          <p className="text-sm text-muted-foreground">
                            {config.metadata.description}
                          </p>
                        )}
                        {renderConfigInput(config)}
                        {config.metadata?.deprecated && (
                          <p className="text-xs text-amber-600">
                            ⚠️ This setting is deprecated
                          </p>
                        )}
                      </div>
                    ))}
                  </CardContent>
                </Card>
              ),
            )}
          </TabsContent>
        )}

        {pluginsWithConfigSections.length > 0 && (
          <TabsContent value="plugins" className="space-y-4">
            <Tabs
              defaultValue={pluginsWithConfigSections[0]?.id}
              className="w-full"
            >
              <TabsList>
                {pluginsWithConfigSections.map((plugin) => (
                  <TabsTrigger key={plugin.id} value={plugin.id}>
                    {plugin.name}
                  </TabsTrigger>
                ))}
              </TabsList>

              {pluginsWithConfigSections.map((plugin) => (
                <TabsContent
                  key={plugin.id}
                  value={plugin.id}
                  className="space-y-4"
                >
                  {plugin.configSection && (
                    <Card>
                      <CardHeader>
                        <CardTitle>{plugin.configSection.title}</CardTitle>
                        {plugin.configSection.description && (
                          <CardDescription>
                            {plugin.configSection.description}
                          </CardDescription>
                        )}
                      </CardHeader>
                      <CardContent className="space-y-6">
                        {plugin.configSection.fields.map((field) => {
                          // Custom fields render their own UI completely
                          if (field.type === "custom") {
                            return (
                              <div key={field.key}>
                                {renderPluginConfigField(field, plugin.id)}
                              </div>
                            );
                          }

                          // Standard fields get label/description wrappers
                          return (
                            <div key={field.key} className="space-y-2">
                              <Label htmlFor={field.key}>
                                {field.label}
                                {field.required && (
                                  <span className="text-red-500 ml-1">*</span>
                                )}
                              </Label>
                              {field.description && (
                                <p className="text-sm text-muted-foreground">
                                  {field.description}
                                </p>
                              )}
                              {renderPluginConfigField(field, plugin.id)}
                              {field.validation?.errorMessage && (
                                <p className="text-xs text-muted-foreground">
                                  {field.validation.errorMessage}
                                </p>
                              )}
                            </div>
                          );
                        })}
                      </CardContent>
                    </Card>
                  )}
                </TabsContent>
              ))}
            </Tabs>
          </TabsContent>
        )}
      </Tabs>
    </div>
  );
}
