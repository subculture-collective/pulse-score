import type { ReactNode } from "react";

interface BaseLayoutProps {
  children: ReactNode;
}

export default function BaseLayout({ children }: BaseLayoutProps) {
  return (
    <div className="min-h-screen bg-gray-50 text-gray-900">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <h1 className="text-xl font-bold text-indigo-600">PulseScore</h1>
          <nav className="flex gap-4 text-sm font-medium text-gray-600">
            <a href="/" className="hover:text-gray-900">
              Dashboard
            </a>
            <a href="/customers" className="hover:text-gray-900">
              Customers
            </a>
            <a href="/settings" className="hover:text-gray-900">
              Settings
            </a>
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {children}
      </main>
    </div>
  );
}
