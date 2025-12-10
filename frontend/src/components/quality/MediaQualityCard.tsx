import { useState } from "react";
import {
  useMediaQuality,
  useQualityProfiles,
  useAssignProfileToMedia,
  useQualityUpgradeHistory,
} from "../../lib/api/quality";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../ui/card";
import { Button } from "../ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select";
import { Badge } from "../ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../ui/table";
import { ArrowUp, Check, Clock, ChevronDown, ChevronUp } from "lucide-react";
import { formatDistanceToNow } from "date-fns";

interface MediaQualityCardProps {
  mediaId: number;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}

export function MediaQualityCard({ mediaId }: MediaQualityCardProps) {
  const { data: mediaQuality, isLoading: qualityLoading } =
    useMediaQuality(mediaId);
  const { data: profiles } = useQualityProfiles();
  const assignProfile = useAssignProfileToMedia(mediaId);
  const { data: history } = useQualityUpgradeHistory(mediaId);

  const [selectedProfileId, setSelectedProfileId] = useState<string>("");
  const [historyExpanded, setHistoryExpanded] = useState(false);

  const handleAssignProfile = async () => {
    if (!selectedProfileId) return;

    try {
      await assignProfile.mutateAsync({
        profile_id: parseInt(selectedProfileId),
      });
      setSelectedProfileId("");
    } catch (error) {
      console.error("Failed to assign profile:", error);
    }
  };

  if (qualityLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Quality Information</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-muted-foreground">
            Loading quality information...
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Quality Information
          {mediaQuality?.upgrade_allowed && !mediaQuality.cutoff_met && (
            <Badge variant="secondary" className="text-xs">
              <ArrowUp className="mr-1 h-3 w-3" />
              Upgradeable
            </Badge>
          )}
          {mediaQuality?.cutoff_met && (
            <Badge variant="default" className="text-xs bg-green-600">
              <Check className="mr-1 h-3 w-3" />
              Cutoff Met
            </Badge>
          )}
        </CardTitle>
        <CardDescription>Current quality and upgrade status</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Current Quality */}
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <div className="text-sm font-medium text-muted-foreground">
                Quality
              </div>
              <div className="text-lg font-semibold">
                {mediaQuality?.quality?.title ||
                  mediaQuality?.detected_quality ||
                  "Unknown"}
              </div>
            </div>

            {mediaQuality?.resolution && (
              <div>
                <div className="text-sm font-medium text-muted-foreground">
                  Resolution
                </div>
                <div className="text-lg font-semibold">
                  {mediaQuality.resolution}p
                </div>
              </div>
            )}

            {mediaQuality?.source && (
              <div>
                <div className="text-sm font-medium text-muted-foreground">
                  Source
                </div>
                <Badge variant="secondary" className="mt-1">
                  {mediaQuality.source}
                </Badge>
              </div>
            )}

            {mediaQuality?.codec_video && (
              <div>
                <div className="text-sm font-medium text-muted-foreground">
                  Video Codec
                </div>
                <Badge variant="outline" className="mt-1">
                  {mediaQuality.codec_video}
                </Badge>
              </div>
            )}

            {mediaQuality?.codec_audio && (
              <div>
                <div className="text-sm font-medium text-muted-foreground">
                  Audio Codec
                </div>
                <Badge variant="outline" className="mt-1">
                  {mediaQuality.codec_audio}
                </Badge>
              </div>
            )}

            {(mediaQuality?.is_proper ||
              mediaQuality?.is_repack ||
              mediaQuality?.is_remux) && (
              <div>
                <div className="text-sm font-medium text-muted-foreground">
                  Modifiers
                </div>
                <div className="flex gap-1 mt-1">
                  {mediaQuality.is_proper && (
                    <Badge variant="secondary">PROPER</Badge>
                  )}
                  {mediaQuality.is_repack && (
                    <Badge variant="secondary">REPACK</Badge>
                  )}
                  {mediaQuality.is_remux && (
                    <Badge variant="secondary">REMUX</Badge>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Quality Profile */}
        <div className="space-y-2">
          <div className="text-sm font-medium">Quality Profile</div>
          {mediaQuality?.profile ? (
            <div className="flex items-center gap-2 p-3 border rounded-md bg-muted">
              <div className="flex-1">
                <div className="font-medium">{mediaQuality.profile.name}</div>
                {mediaQuality.profile.description && (
                  <div className="text-sm text-muted-foreground">
                    {mediaQuality.profile.description}
                  </div>
                )}
              </div>
              {mediaQuality.profile.cutoff_quality && (
                <Badge variant="outline">
                  Cutoff: {mediaQuality.profile.cutoff_quality.title}
                </Badge>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              <div className="text-sm text-muted-foreground">
                No quality profile assigned
              </div>
              <div className="flex gap-2">
                <Select
                  value={selectedProfileId}
                  onValueChange={setSelectedProfileId}
                >
                  <SelectTrigger className="flex-1">
                    <SelectValue placeholder="Select a quality profile" />
                  </SelectTrigger>
                  <SelectContent>
                    {profiles?.map((profile) => (
                      <SelectItem
                        key={profile.id}
                        value={profile.id.toString()}
                      >
                        {profile.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button
                  onClick={handleAssignProfile}
                  disabled={!selectedProfileId || assignProfile.isPending}
                >
                  Assign
                </Button>
              </div>
            </div>
          )}
        </div>

        {/* Upgrade History */}
        {history && history.length > 0 && (
          <div className="space-y-2">
            <Button
              variant="outline"
              className="w-full"
              onClick={() => setHistoryExpanded(!historyExpanded)}
            >
              <Clock className="mr-2 h-4 w-4" />
              Upgrade History ({history.length})
              {historyExpanded ? (
                <ChevronUp className="ml-auto h-4 w-4" />
              ) : (
                <ChevronDown className="ml-auto h-4 w-4" />
              )}
            </Button>

            {historyExpanded && (
              <div className="mt-4">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Date</TableHead>
                      <TableHead>From</TableHead>
                      <TableHead>To</TableHead>
                      <TableHead>Size Change</TableHead>
                      <TableHead>Reason</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {history.map((entry) => (
                      <TableRow key={entry.id}>
                        <TableCell className="text-sm">
                          {formatDistanceToNow(new Date(entry.created_at), {
                            addSuffix: true,
                          })}
                        </TableCell>
                        <TableCell>
                          <Badge variant="secondary">
                            {entry.old_quality?.title || "Unknown"}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant="default">
                            {entry.new_quality?.title || "Unknown"}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-sm">
                          {entry.old_file_size && entry.new_file_size ? (
                            <div className="flex items-center gap-1">
                              <span className="text-muted-foreground">
                                {formatBytes(entry.old_file_size)}
                              </span>
                              →
                              <span className="font-medium">
                                {formatBytes(entry.new_file_size)}
                              </span>
                            </div>
                          ) : (
                            "-"
                          )}
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {entry.reason || "-"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
