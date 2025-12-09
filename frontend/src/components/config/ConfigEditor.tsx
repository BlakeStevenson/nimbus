import { useState } from "react";
import { useConfigValue, useUpdateConfig } from "@/lib/api/config";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Loader2, Pencil } from "lucide-react";
import { formatDate } from "@/lib/utils";

interface ConfigEditorProps {
  configKey: string;
  label: string;
  description?: string;
}

export function ConfigEditor({
  configKey,
  label,
  description,
}: ConfigEditorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [editValue, setEditValue] = useState("");
  const [isJson, setIsJson] = useState(true);

  const { data: config, isLoading, error } = useConfigValue(configKey);
  const updateConfig = useUpdateConfig(configKey);

  const handleEdit = () => {
    if (config) {
      const value = config.value;

      // Check if value is already a string or needs to be JSON stringified
      if (typeof value === "string") {
        setEditValue(value);
        setIsJson(false);
      } else if (typeof value === "object" && value !== null) {
        setEditValue(JSON.stringify(value, null, 2));
        setIsJson(true);
      } else {
        setEditValue(String(value));
        setIsJson(false);
      }
    }
    setIsOpen(true);
  };

  const handleSave = async () => {
    try {
      let parsedValue;
      if (isJson) {
        parsedValue = JSON.parse(editValue);
      } else {
        parsedValue = editValue;
      }

      await updateConfig.mutateAsync(parsedValue);
      setIsOpen(false);
    } catch (error) {
      console.error("Failed to save config:", error);
    }
  };

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{label}</CardTitle>
          {description && <CardDescription>{description}</CardDescription>}
        </CardHeader>
        <CardContent>
          <p className="text-sm text-destructive">
            Failed to load configuration
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>{label}</CardTitle>
            {description && <CardDescription>{description}</CardDescription>}
          </div>
          <Dialog open={isOpen} onOpenChange={setIsOpen}>
            <DialogTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                onClick={handleEdit}
                disabled={isLoading}
              >
                <Pencil className="h-4 w-4 mr-2" />
                Edit
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-2xl">
              <DialogHeader>
                <DialogTitle>Edit {label}</DialogTitle>
                <DialogDescription>
                  Key:{" "}
                  <code className="text-xs bg-muted px-1 py-0.5 rounded">
                    {configKey}
                  </code>
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="value">Value</Label>
                  {isJson ? (
                    <Textarea
                      id="value"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      className="font-mono text-sm min-h-[200px]"
                      placeholder="Enter JSON value..."
                    />
                  ) : (
                    <Input
                      id="value"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      placeholder="Enter value..."
                    />
                  )}
                </div>
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setIsOpen(false)}
                  disabled={updateConfig.isPending}
                >
                  Cancel
                </Button>
                <Button onClick={handleSave} disabled={updateConfig.isPending}>
                  {updateConfig.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Save
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading...
          </div>
        ) : config ? (
          <div className="space-y-2">
            <div className="bg-muted p-3 rounded-md font-mono text-sm overflow-auto">
              {typeof config.value === "string" ? (
                <span>{config.value}</span>
              ) : (
                <pre>{JSON.stringify(config.value, null, 2)}</pre>
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              Last updated: {formatDate(config.updated_at)}
            </p>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">No value set</p>
        )}
      </CardContent>
    </Card>
  );
}
