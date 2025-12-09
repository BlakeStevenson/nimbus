import { useNavigate } from 'react-router-dom';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { MediaKindBadge } from './MediaKindBadge';
import { formatDate } from '@/lib/utils';
import type { MediaItem } from '@/lib/types';

interface MediaTableProps {
  items: MediaItem[];
}

export function MediaTable({ items }: MediaTableProps) {
  const navigate = useNavigate();

  if (items.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        No media items found
      </div>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Title</TableHead>
          <TableHead>Kind</TableHead>
          <TableHead>Year</TableHead>
          <TableHead>Parent</TableHead>
          <TableHead>Updated</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow
            key={item.id}
            className="cursor-pointer"
            onClick={() => navigate(`/media/${item.id}`)}
          >
            <TableCell className="font-medium">{item.title}</TableCell>
            <TableCell>
              <MediaKindBadge kind={item.kind} />
            </TableCell>
            <TableCell>{item.year || '—'}</TableCell>
            <TableCell>
              {item.parent_id ? (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    navigate(`/media/${item.parent_id}`);
                  }}
                  className="text-primary hover:underline"
                >
                  View Parent
                </button>
              ) : (
                '—'
              )}
            </TableCell>
            <TableCell className="text-muted-foreground">
              {formatDate(item.updated_at)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
