import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Topbar } from './Topbar';

export function AppLayout() {
  return (
    <div className="h-screen flex">
      <aside className="w-64 flex-shrink-0">
        <Sidebar />
      </aside>

      <div className="flex-1 flex flex-col overflow-hidden">
        <Topbar />

        <main className="flex-1 overflow-auto">
          <div className="container mx-auto p-6">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
