import { Link, useLocation } from "react-router-dom";
import {
  Home,
  Film,
  Tv,
  Music,
  BookOpen,
  Settings,
  Puzzle,
  Library,
  Users,
  Scan,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { usePluginNavItems } from "@/lib/api/plugins";
import { useAuth } from "@/contexts/AuthContext";

interface NavItem {
  label: string;
  path: string;
  icon?: React.ComponentType<{ className?: string }>;
  group?: string;
}

const coreNavItems: NavItem[] = [
  { label: "Dashboard", path: "/", icon: Home },
  { label: "All Media", path: "/media", icon: Library, group: "Media" },
  { label: "Movies", path: "/media/movies", icon: Film, group: "Media" },
  { label: "TV Shows", path: "/media/tv", icon: Tv, group: "Media" },
  { label: "Music", path: "/media/music", icon: Music, group: "Media" },
  { label: "Books", path: "/media/books", icon: BookOpen, group: "Media" },
];

export function Sidebar() {
  const location = useLocation();
  const pluginNavItems = usePluginNavItems();
  const { user } = useAuth();

  const isActive = (path: string) => {
    if (path === "/") {
      return location.pathname === "/";
    }
    // For /media path, only match exact path or detail pages, not filtered paths
    if (path === "/media") {
      return (
        location.pathname === "/media" ||
        (location.pathname.startsWith("/media/") &&
          /^\/media\/\d+$/.test(location.pathname))
      );
    }
    return location.pathname.startsWith(path);
  };

  const renderNavItems = (items: NavItem[]) => {
    const grouped = items.reduce(
      (acc, item) => {
        const group = item.group || "default";
        if (!acc[group]) acc[group] = [];
        acc[group].push(item);
        return acc;
      },
      {} as Record<string, NavItem[]>,
    );

    return Object.entries(grouped).map(([group, groupItems]) => (
      <div key={group}>
        {group !== "default" && (
          <div className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            {group}
          </div>
        )}
        <div className="space-y-1">
          {groupItems.map((item) => {
            const Icon = item.icon;
            const active = isActive(item.path);

            return (
              <Link
                key={item.path}
                to={item.path}
                className={cn(
                  "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                  active
                    ? "bg-secondary text-secondary-foreground"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                )}
              >
                {Icon && <Icon className="h-4 w-4" />}
                {item.label}
              </Link>
            );
          })}
        </div>
      </div>
    ));
  };

  return (
    <div className="flex flex-col h-full border-r bg-card">
      <div className="p-6">
        <h2 className="text-2xl font-bold">Nimbus</h2>
        <p className="text-sm text-muted-foreground">Media Management</p>
      </div>

      <ScrollArea className="flex-1 px-3">
        <div className="space-y-4 py-4">
          {renderNavItems(coreNavItems)}

          {user?.is_admin && (
            <div>
              <div className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Administration
              </div>
              <div className="space-y-1">
                <Link
                  to="/library"
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                    isActive("/library")
                      ? "bg-secondary text-secondary-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                  )}
                >
                  <Scan className="h-4 w-4" />
                  Library Scanner
                </Link>
                <Link
                  to="/config"
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                    isActive("/config")
                      ? "bg-secondary text-secondary-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                  )}
                >
                  <Settings className="h-4 w-4" />
                  Configuration
                </Link>
                <Link
                  to="/plugins"
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                    isActive("/plugins")
                      ? "bg-secondary text-secondary-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                  )}
                >
                  <Puzzle className="h-4 w-4" />
                  Plugins
                </Link>
                <Link
                  to="/users"
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                    isActive("/users")
                      ? "bg-secondary text-secondary-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                  )}
                >
                  <Users className="h-4 w-4" />
                  Users
                </Link>
              </div>
            </div>
          )}

          {pluginNavItems.length > 0 && (
            <div>
              <div className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Plugin Extensions
              </div>
              <div className="space-y-1">
                {pluginNavItems.map((item) => (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={cn(
                      "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                      isActive(item.path)
                        ? "bg-secondary text-secondary-foreground"
                        : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                    )}
                  >
                    {item.label}
                  </Link>
                ))}
              </div>
            </div>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
