# Nimbus Frontend

A modern React frontend for the Nimbus media management system.

## Tech Stack

- **React 18** - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **Tailwind CSS** - Utility-first CSS
- **shadcn/ui** - Component library
- **TanStack Query** - Data fetching and state management
- **React Router** - Client-side routing

## Getting Started

### Prerequisites

- Node.js 18+ and npm/pnpm/yarn
- A running Nimbus backend server (defaults to `http://localhost:8080`)

### Installation

```bash
# Install dependencies
npm install

# Start development server
npm run dev
```

The app will be available at `http://localhost:5173` by default.

### Build for Production

```bash
npm run build
```

The production build will be output to the `dist/` directory.

## Project Structure

```
src/
├── components/          # React components
│   ├── auth/           # Authentication components (ProtectedRoute)
│   ├── config/         # Configuration components
│   ├── layout/         # Layout components (Sidebar, Topbar, etc.)
│   ├── media/          # Media-related components
│   └── ui/             # shadcn/ui base components
├── contexts/           # React contexts
│   └── AuthContext.tsx # Authentication context
├── lib/                # Core utilities and types
│   ├── api/            # API hooks (TanStack Query)
│   ├── api-client.ts   # Fetch wrapper
│   ├── query-client.ts # TanStack Query client config
│   ├── types.ts        # TypeScript types
│   └── utils.ts        # Utility functions
├── pages/              # Page components
│   ├── DashboardPage.tsx
│   ├── MediaListPage.tsx
│   ├── MediaDetailPage.tsx
│   ├── ConfigPage.tsx
│   ├── PluginsPage.tsx
│   ├── UsersPage.tsx
│   ├── LoginPage.tsx
│   └── RegisterPage.tsx
├── router/             # Router configuration
│   └── routes.tsx
├── App.tsx             # Root component
├── main.tsx            # Entry point
└── index.css           # Global styles
```

## API Integration

The frontend expects a backend API at `/api/*` with the following endpoints:

### Authentication
- `POST /api/auth/login` - Login with username and password
- `POST /api/auth/register` - Register a new user
- `POST /api/auth/logout` - Logout current user
- `GET /api/auth/me` - Get current user info

### Media
- `GET /api/media` - List media items (supports `?kind=`, `?q=`, `?parent_id=` filters)
- `GET /api/media/{id}` - Get media item details
- `PUT /api/media/{id}` - Update media item

### Config
- `GET /api/config/{key}` - Get config value
- `PUT /api/config/{key}` - Update config value

### Users (Admin only)
- `GET /api/users` - List all users
- `GET /api/users/{id}` - Get user details
- `POST /api/users` - Create new user
- `PUT /api/users/{id}` - Update user
- `DELETE /api/users/{id}` - Delete user

### Plugins (Future)
- `GET /api/plugins` - List plugins
- `GET /api/plugins/{id}/ui-manifest` - Get plugin UI manifest

## Plugin Architecture

The frontend is designed to be plugin-ready:

- Plugin navigation items can be injected into the sidebar
- Plugin routes can be dynamically loaded
- The `usePluginUiManifests()` and `usePluginNavItems()` hooks provide integration points
- Currently disabled until backend plugin API is ready

## Configuration

### Vite Proxy

The Vite dev server is configured to proxy `/api/*` requests to `http://localhost:8080` by default. Update `vite.config.ts` to change the backend URL.

### Environment Variables

Create a `.env.local` file to override defaults:

```env
VITE_API_BASE_URL=http://localhost:8080
```

## Development

### Code Style

- ESLint is configured for TypeScript and React
- Run `npm run lint` to check for issues

### Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint

## Features

### Authentication
- User login and registration
- JWT token-based authentication (stored in localStorage)
- Protected routes requiring authentication
- Role-based access control (admin/user)
- User profile dropdown with logout

### Core Features
- Browse media library by type (movies, TV, music, books)
- Search media items
- View and edit media details
- Manage configuration values
- Plugin management (placeholder)

### User Management (Admin Only)
- View all users
- Create new users with roles
- Delete users
- Role-based UI elements (admin features only shown to admins)

### UI Features
- Responsive layout with sidebar navigation
- Clean admin-style interface
- Type-safe API integration
- Optimistic updates with TanStack Query
- Error handling and loading states
- Role-based navigation and features

## Authentication Flow

1. User visits protected route → redirected to `/login`
2. User logs in → JWT token stored in localStorage
3. Token automatically included in all API requests via `Authorization: Bearer {token}` header
4. User data cached in TanStack Query and AuthContext
5. Protected routes check authentication status before rendering
6. Admin-only routes check user role
7. Logout clears token and redirects to login

## Future Enhancements

- [ ] Full plugin support with dynamic routes
- [x] User authentication and management
- [ ] Advanced search and filtering
- [ ] Bulk operations
- [ ] Media file upload
- [ ] Dark mode toggle
- [ ] Accessibility improvements
- [ ] Password reset flow
- [ ] User profile editing
- [ ] Session management and token refresh
