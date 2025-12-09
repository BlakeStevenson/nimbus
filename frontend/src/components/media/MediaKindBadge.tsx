import { Badge } from '@/components/ui/badge';
import { formatMediaKind, getMediaKindColor } from '@/lib/utils';
import type { MediaKind } from '@/lib/types';

interface MediaKindBadgeProps {
  kind: MediaKind;
}

export function MediaKindBadge({ kind }: MediaKindBadgeProps) {
  return (
    <Badge
      variant="outline"
      className={getMediaKindColor(kind)}
    >
      {formatMediaKind(kind)}
    </Badge>
  );
}
