import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  useMediaItem,
  useUpdateMedia,
  useMediaFiles,
  useDeleteMediaFile,
  useDeleteMediaItem,
  useMediaList,
} from "@/lib/api/media";
import { MediaKindBadge } from "@/components/media/MediaKindBadge";
import { MediaGrid } from "@/components/media/MediaGrid";
import { MediaTable } from "@/components/media/MediaTable";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import {
  ArrowLeft,
  Loader2,
  Pencil,
  Trash2,
  X,
  File,
  HardDrive,
  LayoutGrid,
  Table,
} from "lucide-react";
import { formatDate } from "@/lib/utils";

export function MediaDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [deleteConfirmed, setDeleteConfirmed] = useState(false);
  const [isFileDeleteOpen, setIsFileDeleteOpen] = useState(false);
  const [fileToDelete, setFileToDelete] = useState<number | null>(null);
  const [viewMode, setViewMode] = useState<"grid" | "table">("grid");
  const [editData, setEditData] = useState({
    title: "",
    year: "",
    description: "",
  });

  const { data: media, isLoading, error } = useMediaItem(id!);
  const { data: files, isLoading: filesLoading } = useMediaFiles(id!);
  const { data: children, isLoading: childrenLoading } = useMediaList(
    id ? { parentId: Number(id) } : {},
  );
  const { data: parentMedia } = useMediaItem(
    media?.parent_id ? String(media.parent_id) : "",
  );
  const { data: grandparentMedia } = useMediaItem(
    parentMedia?.parent_id ? String(parentMedia.parent_id) : "",
  );
  const updateMedia = useUpdateMedia(id!);
  const deleteFile = useDeleteMediaFile();
  const deleteMediaItem = useDeleteMediaItem();

  const handleEdit = () => {
    if (media) {
      setEditData({
        title: media.title,
        year: media.year?.toString() || "",
        description: (media.metadata?.description as string) || "",
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
      console.error("Failed to update media:", error);
    }
  };

  const handleDeleteFile = (fileId: number) => {
    setFileToDelete(fileId);
    setIsFileDeleteOpen(true);
  };

  const confirmDeleteFile = async (deletePhysical: boolean) => {
    if (fileToDelete === null) return;

    try {
      await deleteFile.mutateAsync({ fileId: fileToDelete, deletePhysical });
      setIsFileDeleteOpen(false);
      setFileToDelete(null);
    } catch (error) {
      console.error("Failed to delete file:", error);
      alert("Failed to delete file");
    }
  };

  const handleDeleteMedia = () => {
    setIsDeleteOpen(true);
    setDeleteConfirmed(false);
  };

  const confirmDeleteMedia = async (deleteFiles: boolean) => {
    if (!deleteConfirmed) {
      alert("Please confirm deletion by checking the box");
      return;
    }

    try {
      await deleteMediaItem.mutateAsync({ mediaId: Number(id), deleteFiles });
      navigate("/media");
    } catch (error) {
      console.error("Failed to delete media item:", error);
      alert("Failed to delete media item");
    }
  };

  const formatFileSize = (bytes: number | null) => {
    if (bytes === null) return "Unknown";
    const units = ["B", "KB", "MB", "GB", "TB"];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${units[unitIndex]}`;
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
        <div className="flex gap-2">
          <Button onClick={handleEdit}>
            <Pencil className="mr-2 h-4 w-4" />
            Edit
          </Button>
          <Button variant="destructive" onClick={handleDeleteMedia}>
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      <div className="space-y-2">
        <div className="flex items-center gap-3 flex-wrap">
          <h1 className="text-3xl font-bold flex items-center gap-2">
            {/* For seasons, show series name */}
            {media.kind === "tv_season" && parentMedia && (
              <>
                <button
                  onClick={() => navigate(`/media/${parentMedia.id}`)}
                  className="text-muted-foreground hover:text-foreground transition-colors"
                >
                  {parentMedia.title}
                </button>
                <span className="text-muted-foreground">/</span>
              </>
            )}
            {/* For episodes, show series name and season */}
            {media.kind === "tv_episode" && grandparentMedia && parentMedia && (
              <>
                <button
                  onClick={() => navigate(`/media/${grandparentMedia.id}`)}
                  className="text-muted-foreground hover:text-foreground transition-colors"
                >
                  {grandparentMedia.title}
                </button>
                <span className="text-muted-foreground">/</span>
                <button
                  onClick={() => navigate(`/media/${parentMedia.id}`)}
                  className="text-muted-foreground hover:text-foreground transition-colors"
                >
                  {parentMedia.title}
                </button>
                <span className="text-muted-foreground">/</span>
              </>
            )}
            <span>
              {media.kind === "tv_episode" &&
                media.metadata &&
                typeof media.metadata === "object" &&
                (media.metadata as Record<string, unknown>).episode && (
                  <span className="text-muted-foreground">
                    E{(media.metadata as Record<string, unknown>).episode}{" "}
                  </span>
                )}
              {media.title}
            </span>
          </h1>
          <MediaKindBadge kind={media.kind} />
        </div>
        {media.year && (
          <p className="text-lg text-muted-foreground">{media.year}</p>
        )}
      </div>

      {/* Children Section */}
      {(childrenLoading ||
        (children && children.items && children.items.length > 0)) && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>
                  {media.kind === "tv_series"
                    ? "Seasons"
                    : media.kind === "tv_season"
                      ? "Episodes"
                      : media.kind === "music_artist"
                        ? "Albums"
                        : media.kind === "music_album"
                          ? "Tracks"
                          : "Children"}
                </CardTitle>
                {children && (
                  <CardDescription>
                    {children.total} {children.total === 1 ? "item" : "items"}
                  </CardDescription>
                )}
              </div>
              {children && children.items && children.items.length > 0 && (
                <div className="flex gap-2">
                  <Button
                    variant={viewMode === "grid" ? "default" : "outline"}
                    size="sm"
                    onClick={() => setViewMode("grid")}
                  >
                    <LayoutGrid className="h-4 w-4 mr-2" />
                    Grid
                  </Button>
                  <Button
                    variant={viewMode === "table" ? "default" : "outline"}
                    size="sm"
                    onClick={() => setViewMode("table")}
                  >
                    <Table className="h-4 w-4 mr-2" />
                    Table
                  </Button>
                </div>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {childrenLoading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : children && children.items && children.items.length > 0 ? (
              <>
                {viewMode === "grid" && <MediaGrid items={children.items} />}
                {viewMode === "table" && <MediaTable items={children.items} />}
              </>
            ) : (
              <p className="text-sm text-muted-foreground">No children found</p>
            )}
          </CardContent>
        </Card>
      )}

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
            <CardTitle>Files</CardTitle>
            <CardDescription>Associated media files</CardDescription>
          </CardHeader>
          <CardContent>
            {filesLoading ? (
              <div className="flex items-center justify-center py-4">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : files && files.length > 0 ? (
              <div className="space-y-2">
                {files.map((file) => (
                  <div
                    key={file.id}
                    className="flex items-start justify-between p-3 rounded-lg bg-muted group"
                  >
                    <div className="flex-1 min-w-0 space-y-1">
                      <div className="flex items-center gap-2">
                        <File className="h-4 w-4 text-muted-foreground shrink-0" />
                        <p
                          className="text-sm font-mono truncate"
                          title={file.path}
                        >
                          {file.path.split("/").pop()}
                        </p>
                      </div>
                      <div className="flex items-center gap-3 text-xs text-muted-foreground">
                        <span>{formatFileSize(file.size)}</span>
                        <span>•</span>
                        <span className="font-mono truncate" title={file.path}>
                          {file.path}
                        </span>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 w-8 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                      onClick={() => handleDeleteFile(file.id)}
                    >
                      <X className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No files</p>
            )}
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
            {media.external_ids &&
            Object.keys(media.external_ids).length > 0 ? (
              <div className="bg-muted p-3 rounded-md font-mono text-sm overflow-auto max-h-96">
                <pre>{JSON.stringify(media.external_ids, null, 2)}</pre>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No external IDs</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Edit Dialog */}
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
                  setEditData((prev) => ({
                    ...prev,
                    description: e.target.value,
                  }))
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

      {/* Delete Media Dialog */}
      <Dialog open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Media Item</DialogTitle>
            <DialogDescription>
              This action cannot be undone. This will permanently delete the
              media item from the database.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="confirm-delete"
                checked={deleteConfirmed}
                onCheckedChange={(checked) =>
                  setDeleteConfirmed(checked as boolean)
                }
              />
              <label
                htmlFor="confirm-delete"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                I understand this action cannot be undone
              </label>
            </div>
            {files && files.length > 0 && (
              <div className="rounded-lg border border-border p-4 space-y-2">
                <p className="text-sm font-medium">
                  This media item has {files.length} associated file(s)
                </p>
                <p className="text-sm text-muted-foreground">
                  Choose whether to keep or delete the physical files:
                </p>
              </div>
            )}
          </div>
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={() => {
                setIsDeleteOpen(false);
                setDeleteConfirmed(false);
              }}
              disabled={deleteMediaItem.isPending}
              className="w-full sm:w-auto"
            >
              Cancel
            </Button>
            <Button
              variant="secondary"
              onClick={() => confirmDeleteMedia(false)}
              disabled={deleteMediaItem.isPending || !deleteConfirmed}
              className="w-full sm:w-auto"
            >
              {deleteMediaItem.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <HardDrive className="mr-2 h-4 w-4" />
              )}
              Delete (Keep Files)
            </Button>
            <Button
              variant="destructive"
              onClick={() => confirmDeleteMedia(true)}
              disabled={deleteMediaItem.isPending || !deleteConfirmed}
              className="w-full sm:w-auto"
            >
              {deleteMediaItem.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Trash2 className="mr-2 h-4 w-4" />
              )}
              Delete (With Files)
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete File Dialog */}
      <Dialog open={isFileDeleteOpen} onOpenChange={setIsFileDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete File</DialogTitle>
            <DialogDescription>
              Choose whether to remove just the database entry or also delete
              the physical file.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={() => {
                setIsFileDeleteOpen(false);
                setFileToDelete(null);
              }}
              disabled={deleteFile.isPending}
              className="w-full sm:w-auto"
            >
              Cancel
            </Button>
            <Button
              variant="secondary"
              onClick={() => confirmDeleteFile(false)}
              disabled={deleteFile.isPending}
              className="w-full sm:w-auto"
            >
              {deleteFile.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <HardDrive className="mr-2 h-4 w-4" />
              )}
              Remove Entry Only
            </Button>
            <Button
              variant="destructive"
              onClick={() => confirmDeleteFile(true)}
              disabled={deleteFile.isPending}
              className="w-full sm:w-auto"
            >
              {deleteFile.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Trash2 className="mr-2 h-4 w-4" />
              )}
              Delete from Disk
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
