import { useState } from "react";
import { useUsers, useCreateUser, useDeleteUser } from "@/lib/api/auth";
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
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Loader2, Plus, Trash2 } from "lucide-react";
import { formatDate } from "@/lib/utils";
import type { CreateUserRequest } from "@/lib/types";

export function UsersPage() {
  const { data: usersData, isLoading, error } = useUsers();
  const createUser = useCreateUser();
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [formData, setFormData] = useState<CreateUserRequest>({
    username: "",
    email: "",
    password: "",
    is_admin: false,
  });
  const [deleteId, setDeleteId] = useState<string | number | null>(null);
  const deleteUser = useDeleteUser(deleteId!);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createUser.mutateAsync(formData);
      setIsCreateOpen(false);
      setFormData({ username: "", email: "", password: "", is_admin: false });
    } catch (error) {
      console.error("Failed to create user:", error);
    }
  };

  const handleDelete = async (id: string | number) => {
    if (confirm("Are you sure you want to delete this user?")) {
      setDeleteId(id);
      try {
        await deleteUser.mutateAsync();
      } catch (error) {
        console.error("Failed to delete user:", error);
      }
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">User Management</h1>
          <p className="text-muted-foreground">
            Manage user accounts and permissions
          </p>
        </div>
        <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Create User
            </Button>
          </DialogTrigger>
          <DialogContent>
            <form onSubmit={handleCreate}>
              <DialogHeader>
                <DialogTitle>Create New User</DialogTitle>
                <DialogDescription>
                  Add a new user to the system
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="username">Username</Label>
                  <Input
                    id="username"
                    value={formData.username}
                    onChange={(e) =>
                      setFormData({ ...formData, username: e.target.value })
                    }
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    value={formData.email}
                    onChange={(e) =>
                      setFormData({ ...formData, email: e.target.value })
                    }
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="password">Password</Label>
                  <Input
                    id="password"
                    type="password"
                    value={formData.password}
                    onChange={(e) =>
                      setFormData({ ...formData, password: e.target.value })
                    }
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="role">Role</Label>
                  <Select
                    value={formData.is_admin ? "admin" : "user"}
                    onValueChange={(value) =>
                      setFormData({ ...formData, is_admin: value === "admin" })
                    }
                  >
                    <SelectTrigger id="role">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="user">User</SelectItem>
                      <SelectItem value="admin">Admin</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setIsCreateOpen(false)}
                  disabled={createUser.isPending}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={createUser.isPending}>
                  {createUser.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Create
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Users</CardTitle>
          <CardDescription>
            {usersData
              ? `${usersData.total} ${usersData.total === 1 ? "user" : "users"}`
              : "Loading..."}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {error && (
            <p className="text-sm text-destructive">Failed to load users</p>
          )}

          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          )}

          {usersData && usersData.users.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              No users found
            </div>
          )}

          {usersData && usersData.users.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Username</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usersData.users.map((user) => (
                  <TableRow key={user.id}>
                    <TableCell className="font-medium">
                      {user.username}
                    </TableCell>
                    <TableCell>{user.email || "—"}</TableCell>
                    <TableCell>
                      <Badge variant={user.is_admin ? "default" : "secondary"}>
                        {user.is_admin ? "Admin" : "User"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDate(user.created_at)}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDelete(user.id)}
                        disabled={deleteUser.isPending && deleteId === user.id}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
