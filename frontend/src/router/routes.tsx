import { createBrowserRouter } from "react-router-dom";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { AppLayout } from "@/components/layout/AppLayout";
import { DashboardPage } from "@/pages/DashboardPage";
import { MediaListPage } from "@/pages/MediaListPage";
import { MediaDetailPage } from "@/pages/MediaDetailPage";
import { ConfigPage } from "@/pages/ConfigPage";
import { PluginsPage } from "@/pages/PluginsPage";
import { UsersPage } from "@/pages/UsersPage";
import { LoginPage } from "@/pages/LoginPage";
import { RegisterPage } from "@/pages/RegisterPage";

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/register",
    element: <RegisterPage />,
  },
  {
    path: "/",
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      {
        index: true,
        element: <DashboardPage />,
      },
      {
        path: "media",
        element: (
          <MediaListPage
            title="All Media"
            description="Browse all media items"
          />
        ),
      },
      {
        path: "media/movies",
        element: (
          <MediaListPage
            defaultKind="movie"
            title="Movies"
            description="Browse your movie collection"
          />
        ),
      },
      {
        path: "media/tv",
        element: (
          <MediaListPage
            defaultKind="tv_series"
            title="TV Shows"
            description="Browse your TV series collection"
          />
        ),
      },
      {
        path: "media/music",
        element: (
          <MediaListPage
            defaultKind="music_album"
            title="Music"
            description="Browse your music collection"
          />
        ),
      },
      {
        path: "media/books",
        element: (
          <MediaListPage
            defaultKind="book"
            title="Books"
            description="Browse your book collection"
          />
        ),
      },
      {
        path: "media/:id",
        element: <MediaDetailPage />,
      },
      {
        path: "config",
        element: <ConfigPage />,
      },
      {
        path: "plugins",
        element: <PluginsPage />,
      },
      {
        path: "users",
        element: (
          <ProtectedRoute requireAdmin>
            <UsersPage />
          </ProtectedRoute>
        ),
      },
    ],
  },
]);
