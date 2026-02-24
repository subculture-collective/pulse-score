import { Link, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  Users,
  Plug,
  Settings,
  ChevronLeft,
  ChevronRight,
  Menu,
  X,
} from "lucide-react";

const navItems = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard },
  { to: "/customers", label: "Customers", icon: Users },
  { to: "/settings/integrations", label: "Integrations", icon: Plug },
  { to: "/settings", label: "Settings", icon: Settings, end: true },
];

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  mobileOpen: boolean;
  onCloseMobile: () => void;
}

export default function Sidebar({
  collapsed,
  onToggleCollapse,
  mobileOpen,
  onCloseMobile,
}: SidebarProps) {
  const location = useLocation();

  function isActive(to: string, end?: boolean) {
    if (end) return location.pathname === to;
    if (to === "/") return location.pathname === "/";
    return location.pathname.startsWith(to);
  }

  const nav = (
    <nav className="flex flex-1 flex-col gap-1 px-3 py-4">
      {navItems.map((item) => {
        const active = isActive(item.to, item.end);
        const Icon = item.icon;
        return (
          <Link
            key={item.to}
            to={item.to}
            onClick={onCloseMobile}
            title={collapsed ? item.label : undefined}
            className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
              active
                ? "bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300"
                : "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800"
            } ${collapsed ? "justify-center" : ""}`}
          >
            <Icon className="h-5 w-5 shrink-0" />
            {!collapsed && <span>{item.label}</span>}
          </Link>
        );
      })}
    </nav>
  );

  return (
    <>
      {/* Mobile overlay */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={onCloseMobile}
        />
      )}

      {/* Mobile drawer */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-gray-200 bg-white transition-transform duration-300 dark:border-gray-700 dark:bg-gray-900 md:hidden ${
          mobileOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <div className="flex h-16 items-center justify-between border-b border-gray-200 px-4 dark:border-gray-700">
          <span className="text-lg font-bold text-indigo-600 dark:text-indigo-400">
            PulseScore
          </span>
          <button
            onClick={onCloseMobile}
            className="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        {nav}
      </aside>

      {/* Desktop sidebar */}
      <aside
        className={`hidden flex-col border-r border-gray-200 bg-white transition-all duration-300 dark:border-gray-700 dark:bg-gray-900 md:flex ${
          collapsed ? "w-16" : "w-64"
        }`}
      >
        <div
          className={`flex h-16 items-center border-b border-gray-200 dark:border-gray-700 ${
            collapsed ? "justify-center px-2" : "px-4"
          }`}
        >
          {!collapsed && (
            <span className="text-lg font-bold text-indigo-600 dark:text-indigo-400">
              PulseScore
            </span>
          )}
          {collapsed && (
            <span className="text-lg font-bold text-indigo-600 dark:text-indigo-400">
              PS
            </span>
          )}
        </div>
        {nav}
        <div className="border-t border-gray-200 p-3 dark:border-gray-700">
          <button
            onClick={onToggleCollapse}
            className="flex w-full items-center justify-center rounded-lg p-2 text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
            title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <ChevronLeft className="h-4 w-4" />
            )}
          </button>
        </div>
      </aside>

      {/* Mobile hamburger (rendered in header area but controlled here) */}
      <button
        onClick={() => (mobileOpen ? onCloseMobile() : onToggleCollapse())}
        className="fixed left-4 top-4 z-30 rounded-lg bg-white p-2 shadow-md dark:bg-gray-800 md:hidden"
        aria-label="Toggle navigation"
      >
        <Menu className="h-5 w-5 text-gray-700 dark:text-gray-300" />
      </button>
    </>
  );
}
