import { useState, useRef, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Bell } from "lucide-react";
import { notificationsApi, type AppNotification } from "@/lib/api";

const POLL_INTERVAL = 30_000;

export default function NotificationBell() {
  const [unreadCount, setUnreadCount] = useState(0);
  const [notifications, setNotifications] = useState<AppNotification[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  const fetchCount = useCallback(async () => {
    try {
      const { data } = await notificationsApi.unreadCount();
      setUnreadCount(data.count);
    } catch {
      // Silently fail — don't block UI
    }
  }, []);

  // Poll unread count
  useEffect(() => {
    fetchCount();
    const timer = setInterval(fetchCount, POLL_INTERVAL);
    return () => clearInterval(timer);
  }, [fetchCount]);

  // Close on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  async function toggleOpen() {
    if (!open) {
      setLoading(true);
      try {
        const { data } = await notificationsApi.list({ limit: 10 });
        setNotifications(data.notifications ?? []);
      } catch {
        setNotifications([]);
      } finally {
        setLoading(false);
      }
    }
    setOpen(!open);
  }

  async function handleMarkRead(id: string) {
    await notificationsApi.markRead(id);
    setNotifications((prev) =>
      prev.map((n) =>
        n.id === id ? { ...n, read_at: new Date().toISOString() } : n,
      ),
    );
    setUnreadCount((c) => Math.max(0, c - 1));
  }

  async function handleMarkAllRead() {
    await notificationsApi.markAllRead();
    setNotifications((prev) =>
      prev.map((n) => ({
        ...n,
        read_at: n.read_at ?? new Date().toISOString(),
      })),
    );
    setUnreadCount(0);
  }

  function handleNotificationClick(n: AppNotification) {
    if (!n.read_at) handleMarkRead(n.id);
    const customerId = (n.data as Record<string, string>)?.customer_id;
    if (customerId) {
      navigate(`/customers/${customerId}`);
      setOpen(false);
    }
  }

  function timeAgo(dateStr: string): string {
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    return `${Math.floor(hrs / 24)}d ago`;
  }

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={toggleOpen}
        className="relative rounded-lg p-2 text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
        aria-label="Notifications"
      >
        <Bell className="h-5 w-5" />
        {unreadCount > 0 && (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
            {unreadCount > 99 ? "99+" : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-full z-50 mt-2 w-80 rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800">
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-gray-700">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
              Notifications
            </h3>
            {unreadCount > 0 && (
              <button
                onClick={handleMarkAllRead}
                className="text-xs text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300"
              >
                Mark all read
              </button>
            )}
          </div>

          <div className="max-h-80 overflow-y-auto">
            {loading ? (
              <div className="px-4 py-6 text-center text-sm text-gray-500 dark:text-gray-400">
                Loading...
              </div>
            ) : notifications.length === 0 ? (
              <div className="px-4 py-6 text-center text-sm text-gray-500 dark:text-gray-400">
                No notifications
              </div>
            ) : (
              notifications.map((n) => (
                <button
                  key={n.id}
                  onClick={() => handleNotificationClick(n)}
                  className={`block w-full px-4 py-3 text-left transition-colors hover:bg-gray-50 dark:hover:bg-gray-700/50 ${
                    !n.read_at ? "bg-indigo-50/50 dark:bg-indigo-900/10" : ""
                  }`}
                >
                  <div className="flex items-start gap-2">
                    {!n.read_at && (
                      <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-indigo-500" />
                    )}
                    <div className={!n.read_at ? "" : "ml-4"}>
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        {n.title}
                      </p>
                      <p className="mt-0.5 text-xs text-gray-500 dark:text-gray-400 line-clamp-2">
                        {n.message}
                      </p>
                      <p className="mt-1 text-[10px] text-gray-400 dark:text-gray-500">
                        {timeAgo(n.created_at)}
                      </p>
                    </div>
                  </div>
                </button>
              ))
            )}
          </div>

          <div className="border-t border-gray-200 dark:border-gray-700">
            <button
              onClick={() => {
                setOpen(false);
                navigate("/settings/notifications");
              }}
              className="block w-full px-4 py-2.5 text-center text-xs text-indigo-600 hover:bg-gray-50 dark:text-indigo-400 dark:hover:bg-gray-700/50"
            >
              Notification settings
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
