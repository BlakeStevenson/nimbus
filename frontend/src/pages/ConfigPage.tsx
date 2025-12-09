import { useState, useEffect } from "react";
import { useAllConfig, useUpdateMultipleConfigs } from "@/lib/api/config";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
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
import { Loader2, Save } from "lucide-react";
import type { ConfigValue } from "@/lib/types";

export function ConfigPage() {
  const { data: configs, isLoading, error } = useAllConfig();
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
        // For multi-select, we'll use checkboxes
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

  if (error) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Configuration</h1>
          <p className="text-muted-foreground">
            Manage system configuration settings
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

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Configuration</h1>
          <p className="text-muted-foreground">
            Manage system configuration settings
          </p>
        </div>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Configuration</h1>
          <p className="text-muted-foreground">
            Manage system configuration settings
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

      <div className="space-y-4">
        {configs && configs.length > 0 ? (
          configs.map((config) => (
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
                No configuration settings found
              </p>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
