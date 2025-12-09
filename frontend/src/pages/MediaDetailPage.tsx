import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useMediaItem, useUpdateMedia } from '@/lib/api/media';
import { MediaKindBadge } from '@/components/media/MediaKindBadge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { ArrowLeft, Loader2, Pencil } from 'lucide-react';
import { formatDate } from '@/lib/utils';

export function MediaDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [editData, setEditData] = useState({
    title: '',
    year: '',
    description: '',
  });

  const { data: media, isLoading, error } = useMediaItem(id!);
  const updateMedia = useUpdateMedia(id!);

  const handleEdit = () => {
    if (media) {
      setEditData({
        title: media.title,
        year: media.year?.toString() || '',
        description: (media.metadata?.description as string) || '',
      });
      setIsEditOpen(true);
    }
  };

  const handleSave = async () => {
    try {
      await updateMedia.mutateAsync({
        title: editData.title,
        year: editData.year ? parseInt(editData.year) : null,
        metadata: {
          ...media?.metadata,
          description: editData.description,
        },
      });
      setIsEditOpen(false);
    } catch (error) {
      console.error('Failed to update media:', error);
    }
  };

  if (error) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" onClick={() => navigate(-1)}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-destructive">
              Failed to load media item
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!media) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" onClick={() => navigate(-1)}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">
              Media item not found
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Button variant="ghost" onClick={() => navigate(-1)}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Button onClick={handleEdit}>
          <Pencil className="mr-2 h-4 w-4" />
          Edit
        </Button>
      </div>

      <div className="space-y-2">
        <div className="flex items-center gap-3">
          <h1 className="text-3xl font-bold">{media.title}</h1>
          <MediaKindBadge kind={media.kind} />
        </div>
        {media.year && (
          <p className="text-lg text-muted-foreground">{media.year}</p>
        )}
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Details</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <Label className="text-muted-foreground">ID</Label>
              <p className="font-mono text-sm">{media.id}</p>
            </div>
            <div>
              <Label className="text-muted-foreground">Sort Title</Label>
              <p>{media.sort_title}</p>
            </div>
            {media.parent_id && (
              <div>
                <Label className="text-muted-foreground">Parent</Label>
                <Button
                  variant="link"
                  className="h-auto p-0"
                  onClick={() => navigate(`/media/${media.parent_id}`)}
                >
                  View Parent
                </Button>
              </div>
            )}
            <div>
              <Label className="text-muted-foreground">Created</Label>
              <p className="text-sm">{formatDate(media.created_at)}</p>
            </div>
            <div>
              <Label className="text-muted-foreground">Updated</Label>
              <p className="text-sm">{formatDate(media.updated_at)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Metadata</CardTitle>
            <CardDescription>Custom metadata fields</CardDescription>
          </CardHeader>
          <CardContent>
            {media.metadata && Object.keys(media.metadata).length > 0 ? (
              <div className="bg-muted p-3 rounded-md font-mono text-sm overflow-auto max-h-96">
                <pre>{JSON.stringify(media.metadata, null, 2)}</pre>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No metadata</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>External IDs</CardTitle>
            <CardDescription>External service identifiers</CardDescription>
          </CardHeader>
          <CardContent>
            {media.external_ids && Object.keys(media.external_ids).length > 0 ? (
              <div className="bg-muted p-3 rounded-md font-mono text-sm overflow-auto max-h-96">
                <pre>{JSON.stringify(media.external_ids, null, 2)}</pre>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No external IDs</p>
            )}
          </CardContent>
        </Card>
      </div>

      <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Media Item</DialogTitle>
            <DialogDescription>
              Update basic information for this media item
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="title">Title</Label>
              <Input
                id="title"
                value={editData.title}
                onChange={(e) =>
                  setEditData((prev) => ({ ...prev, title: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="year">Year</Label>
              <Input
                id="year"
                type="number"
                value={editData.year}
                onChange={(e) =>
                  setEditData((prev) => ({ ...prev, year: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={editData.description}
                onChange={(e) =>
                  setEditData((prev) => ({ ...prev, description: e.target.value }))
                }
                rows={4}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsEditOpen(false)}
              disabled={updateMedia.isPending}
            >
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={updateMedia.isPending}>
              {updateMedia.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
