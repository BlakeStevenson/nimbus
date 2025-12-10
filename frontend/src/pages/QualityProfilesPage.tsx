import { useState } from "react";
import {
  useQualityProfiles,
  useDeleteQualityProfile,
} from "../lib/api/quality";
import { Button } from "../components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import { Badge } from "../components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Plus, Pencil, Trash2, ArrowUp, Check } from "lucide-react";
import { QualityProfileDialog } from "../components/quality/QualityProfileDialog";
import type { QualityProfile } from "../lib/types/quality";

export function QualityProfilesPage() {
  const { data: profiles, isLoading } = useQualityProfiles();
  const deleteProfile = useDeleteQualityProfile();

  const [selectedProfile, setSelectedProfile] = useState<QualityProfile | null>(
    null,
  );
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [profileToDelete, setProfileToDelete] = useState<QualityProfile | null>(
    null,
  );

  const handleCreate = () => {
    setSelectedProfile(null);
    setIsDialogOpen(true);
  };

  const handleEdit = (profile: QualityProfile) => {
    setSelectedProfile(profile);
    setIsDialogOpen(true);
  };

  const handleDelete = (profile: QualityProfile) => {
    setProfileToDelete(profile);
  };

  const confirmDelete = async () => {
    if (profileToDelete) {
      await deleteProfile.mutateAsync(profileToDelete.id);
      setProfileToDelete(null);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-muted-foreground">Loading quality profiles...</div>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Quality Profiles</h1>
          <p className="text-muted-foreground mt-1">
            Manage quality preferences for your media library
          </p>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" />
          New Profile
        </Button>
      </div>

      <div className="grid gap-6">
        {profiles?.map((profile) => (
          <Card key={profile.id}>
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="space-y-1">
                  <CardTitle className="flex items-center gap-2">
                    {profile.name}
                    {profile.upgrade_allowed && (
                      <Badge variant="secondary" className="text-xs">
                        <ArrowUp className="mr-1 h-3 w-3" />
                        Upgrades Enabled
                      </Badge>
                    )}
                  </CardTitle>
                  {profile.description && (
                    <CardDescription>{profile.description}</CardDescription>
                  )}
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleEdit(profile)}
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleDelete(profile)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {profile.cutoff_quality && (
                <div className="mb-4 p-3 bg-muted rounded-md">
                  <div className="flex items-center gap-2 text-sm">
                    <Check className="h-4 w-4 text-green-600" />
                    <span className="font-medium">Cutoff Quality:</span>
                    <Badge variant="outline">
                      {profile.cutoff_quality.title}
                    </Badge>
                  </div>
                </div>
              )}

              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Quality</TableHead>
                    <TableHead>Resolution</TableHead>
                    <TableHead>Source</TableHead>
                    <TableHead>Weight</TableHead>
                    <TableHead>Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {profile.items
                    ?.sort((a, b) => a.sort_order - b.sort_order)
                    .map((item) => (
                      <TableRow key={item.id}>
                        <TableCell className="font-medium">
                          {item.quality?.title}
                        </TableCell>
                        <TableCell>
                          {item.quality?.resolution
                            ? `${item.quality.resolution}p`
                            : "-"}
                        </TableCell>
                        <TableCell>
                          {item.quality?.source ? (
                            <Badge variant="secondary">
                              {item.quality.source}
                            </Badge>
                          ) : (
                            "-"
                          )}
                        </TableCell>
                        <TableCell>{item.quality?.weight}</TableCell>
                        <TableCell>
                          {item.allowed ? (
                            <Badge className="bg-green-600">Allowed</Badge>
                          ) : (
                            <Badge variant="secondary">Blocked</Badge>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        ))}
      </div>

      <QualityProfileDialog
        key={selectedProfile?.id || "new"}
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        profile={selectedProfile}
      />

      <Dialog
        open={!!profileToDelete}
        onOpenChange={(open) => !open && setProfileToDelete(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Quality Profile</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{profileToDelete?.name}"? This
              action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setProfileToDelete(null)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={confirmDelete}>
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
