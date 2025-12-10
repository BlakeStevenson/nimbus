import { useState } from "react";
import {
  useQualityDefinitions,
  useCreateQualityProfile,
  useUpdateQualityProfile,
} from "../../lib/api/quality";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Label } from "../ui/label";
import { Textarea } from "../ui/textarea";
import { Checkbox } from "../ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select";

import { Badge } from "../ui/badge";
import { GripVertical } from "lucide-react";
import type {
  QualityProfile,
  CreateQualityProfileParams,
  CreateQualityProfileItemParams,
} from "../../lib/types/quality";

interface QualityProfileDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  profile?: QualityProfile | null;
}

function getInitialFormState(profile?: QualityProfile | null) {
  if (profile) {
    const qualityMap = new Map<
      number,
      { allowed: boolean; sortOrder: number }
    >();
    profile.items?.forEach((item) => {
      qualityMap.set(item.quality_id, {
        allowed: item.allowed,
        sortOrder: item.sort_order,
      });
    });

    return {
      name: profile.name,
      description: profile.description || "",
      cutoffQualityId: profile.cutoff_quality_id?.toString() || "",
      upgradeAllowed: profile.upgrade_allowed,
      selectedQualities: qualityMap,
    };
  }

  return {
    name: "",
    description: "",
    cutoffQualityId: "",
    upgradeAllowed: true,
    selectedQualities: new Map<
      number,
      { allowed: boolean; sortOrder: number }
    >(),
  };
}

export function QualityProfileDialog({
  open,
  onOpenChange,
  profile,
}: QualityProfileDialogProps) {
  const { data: definitions } = useQualityDefinitions();
  const createProfile = useCreateQualityProfile();
  const updateProfile = useUpdateQualityProfile(profile?.id || 0);

  const initialState = getInitialFormState(profile);
  const [name, setName] = useState(initialState.name);
  const [description, setDescription] = useState(initialState.description);
  const [cutoffQualityId, setCutoffQualityId] = useState(
    initialState.cutoffQualityId,
  );
  const [upgradeAllowed, setUpgradeAllowed] = useState(
    initialState.upgradeAllowed,
  );
  const [selectedQualities, setSelectedQualities] = useState(
    initialState.selectedQualities,
  );

  const toggleQuality = (qualityId: number) => {
    setSelectedQualities((prev) => {
      const newMap = new Map(prev);
      if (newMap.has(qualityId)) {
        newMap.delete(qualityId);
      } else {
        newMap.set(qualityId, {
          allowed: true,
          sortOrder: newMap.size,
        });
      }
      return newMap;
    });
  };

  const toggleAllowed = (qualityId: number) => {
    setSelectedQualities((prev) => {
      const newMap = new Map(prev);
      const current = newMap.get(qualityId);
      if (current) {
        newMap.set(qualityId, {
          ...current,
          allowed: !current.allowed,
        });
      }
      return newMap;
    });
  };

  const moveQuality = (qualityId: number, direction: "up" | "down") => {
    setSelectedQualities((prev) => {
      const entries = Array.from(prev.entries()).sort(
        (a, b) => a[1].sortOrder - b[1].sortOrder,
      );
      const index = entries.findIndex(([id]) => id === qualityId);

      if (
        (direction === "up" && index === 0) ||
        (direction === "down" && index === entries.length - 1)
      ) {
        return prev;
      }

      const swapIndex = direction === "up" ? index - 1 : index + 1;
      [entries[index], entries[swapIndex]] = [
        entries[swapIndex],
        entries[index],
      ];

      const newMap = new Map<number, { allowed: boolean; sortOrder: number }>();
      entries.forEach(([id, data], idx) => {
        newMap.set(id, { ...data, sortOrder: idx });
      });

      return newMap;
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const items: CreateQualityProfileItemParams[] = Array.from(
      selectedQualities.entries(),
    )
      .sort((a, b) => a[1].sortOrder - b[1].sortOrder)
      .map(([qualityId, { allowed, sortOrder }]) => ({
        quality_id: qualityId,
        allowed,
        sort_order: sortOrder,
      }));

    const params: CreateQualityProfileParams = {
      name,
      description: description || undefined,
      cutoff_quality_id: cutoffQualityId
        ? parseInt(cutoffQualityId)
        : undefined,
      upgrade_allowed: upgradeAllowed,
      items,
    };

    try {
      if (profile) {
        await updateProfile.mutateAsync(params);
      } else {
        await createProfile.mutateAsync(params);
      }
      onOpenChange(false);
    } catch (error) {
      console.error("Failed to save profile:", error);
    }
  };

  const sortedDefinitions = definitions
    ?.slice()
    .sort((a, b) => b.weight - a.weight);

  const selectedQualitiesArray = Array.from(selectedQualities.entries())
    .sort((a, b) => a[1].sortOrder - b[1].sortOrder)
    .map(([id, data]) => ({
      id,
      ...data,
      definition: definitions?.find((d) => d.id === id),
    }));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] flex flex-col overflow-hidden">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle>
            {profile ? "Edit Quality Profile" : "Create Quality Profile"}
          </DialogTitle>
          <DialogDescription>
            Configure quality preferences and upgrade rules for your media.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
          <div className="overflow-y-auto flex-1 pr-4">
            <div className="space-y-6 py-4">
              <div className="space-y-2">
                <Label htmlFor="name">Profile Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="HD-1080p"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Prefer 1080p WEB-DL or Bluray with upgrades allowed"
                />
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="upgrade_allowed"
                  checked={upgradeAllowed}
                  onCheckedChange={(checked) =>
                    setUpgradeAllowed(checked === true)
                  }
                />
                <Label htmlFor="upgrade_allowed" className="cursor-pointer">
                  Allow automatic quality upgrades
                </Label>
              </div>

              {upgradeAllowed && (
                <div className="space-y-2">
                  <Label htmlFor="cutoff_quality_id">Cutoff Quality</Label>
                  <Select
                    value={cutoffQualityId}
                    onValueChange={(value) => setCutoffQualityId(value)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select cutoff quality" />
                    </SelectTrigger>
                    <SelectContent>
                      {sortedDefinitions?.map((def) => (
                        <SelectItem key={def.id} value={def.id.toString()}>
                          {def.title}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">
                    Stop upgrading once this quality is reached
                  </p>
                </div>
              )}

              <div className="space-y-4">
                <Label>Quality Selection</Label>
                <div className="grid grid-cols-2 gap-2">
                  {sortedDefinitions?.map((def) => (
                    <div
                      key={def.id}
                      className={`flex items-center space-x-2 p-2 border rounded-md cursor-pointer hover:bg-muted transition-colors ${
                        selectedQualities.has(def.id)
                          ? "bg-muted border-primary"
                          : ""
                      }`}
                      onClick={() => toggleQuality(def.id)}
                    >
                      <Checkbox
                        checked={selectedQualities.has(def.id)}
                        onCheckedChange={() => toggleQuality(def.id)}
                      />
                      <div className="flex-1">
                        <div className="font-medium text-sm">{def.title}</div>
                        <div className="text-xs text-muted-foreground">
                          Weight: {def.weight}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {selectedQualitiesArray.length > 0 && (
                <div className="space-y-4">
                  <Label>Quality Order (Higher = Better)</Label>
                  <div className="space-y-2">
                    {selectedQualitiesArray.map((item, index) => (
                      <div
                        key={item.id}
                        className="flex items-center gap-2 p-3 border rounded-md bg-card"
                      >
                        <div className="flex flex-col gap-1">
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="h-4 w-4 p-0"
                            onClick={() => moveQuality(item.id, "up")}
                            disabled={index === 0}
                          >
                            ↑
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="h-4 w-4 p-0"
                            onClick={() => moveQuality(item.id, "down")}
                            disabled={
                              index === selectedQualitiesArray.length - 1
                            }
                          >
                            ↓
                          </Button>
                        </div>
                        <GripVertical className="h-4 w-4 text-muted-foreground" />
                        <div className="flex-1">
                          <div className="font-medium">
                            {item.definition?.title || "Unknown"}
                          </div>
                          <div className="text-xs text-muted-foreground">
                            {item.definition?.resolution
                              ? `${item.definition.resolution}p`
                              : "Unknown"}{" "}
                            · {item.definition?.source || "Unknown"}
                          </div>
                        </div>
                        <Checkbox
                          checked={item.allowed}
                          onCheckedChange={() => toggleAllowed(item.id)}
                        />
                        <Badge variant={item.allowed ? "default" : "secondary"}>
                          {item.allowed ? "Allowed" : "Blocked"}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          <DialogFooter className="flex-shrink-0 mt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createProfile.isPending || updateProfile.isPending}
            >
              {profile ? "Update Profile" : "Create Profile"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
