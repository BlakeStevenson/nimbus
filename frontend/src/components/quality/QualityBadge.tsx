import { Badge } from "../ui/badge";
import { TrendingUp, Check, Minus, Ban } from "lucide-react";
import type { DetectedQualityInfo } from "../../lib/types/quality";

interface QualityBadgeProps {
  quality: string;
  isUpgrade?: boolean;
  isCurrent?: boolean;
  isAllowed?: boolean;
  className?: string;
}

export function QualityBadge({
  quality,
  isUpgrade,
  isCurrent,
  isAllowed,
  className,
}: QualityBadgeProps) {
  let variant: "default" | "secondary" | "outline" | "destructive" = "secondary";
  let icon = null;

  if (isUpgrade) {
    variant = "default";
    icon = <TrendingUp className="mr-1 h-3 w-3" />;
  } else if (isCurrent) {
    variant = "outline";
    icon = <Check className="mr-1 h-3 w-3" />;
  } else if (isAllowed === false) {
    variant = "destructive";
    icon = <Ban className="mr-1 h-3 w-3" />;
  }

  return (
    <Badge variant={variant} className={className}>
      {icon}
      {quality}
    </Badge>
  );
}

export function getQualityBadgeFromTitle(title: string): string {
  const lower = title.toLowerCase();
  if (lower.includes("2160p") || lower.includes("4k")) return "2160p";
  if (lower.includes("1080p")) return "1080p";
  if (lower.includes("720p")) return "720p";
  if (lower.includes("480p")) return "480p";
  return "Unknown";
}
