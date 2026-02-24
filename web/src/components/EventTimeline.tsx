import { useCallback, useEffect, useState } from "react";
import api from "@/lib/api";
import EmptyState from "@/components/EmptyState";
import {
  CheckCircle,
  XCircle,
  PlusCircle,
  Flag,
  Circle,
  Loader2,
} from "lucide-react";

interface CustomerEvent {
  id: string;
  type: string;
  title: string;
  source: string;
  timestamp: string;
  data?: Record<string, unknown>;
}

interface EventsResponse {
  events: CustomerEvent[];
  total: number;
  page: number;
  per_page: number;
}

const eventIcons: Record<string, { icon: typeof CheckCircle; color: string }> =
  {
    "payment.success": { icon: CheckCircle, color: "text-green-500" },
    "payment.succeeded": { icon: CheckCircle, color: "text-green-500" },
    "payment.failed": { icon: XCircle, color: "text-red-500" },
    "subscription.created": { icon: PlusCircle, color: "text-blue-500" },
    "subscription.updated": { icon: PlusCircle, color: "text-blue-500" },
    "ticket.opened": { icon: Flag, color: "text-yellow-500" },
  };

function relativeTime(dateStr: string): string {
  if (!dateStr) return "—";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "Just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(dateStr).toLocaleDateString();
}

export default function EventTimeline({ customerId }: { customerId: string }) {
  const [events, setEvents] = useState<CustomerEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const fetchEvents = useCallback(
    async (p: number, append: boolean) => {
      if (append) setLoadingMore(true);
      else setLoading(true);
      try {
        const { data } = await api.get<EventsResponse>(
          `/customers/${customerId}/events`,
          { params: { page: p, per_page: 20 } },
        );
        const evts = data.events ?? [];
        setEvents((prev) => (append ? [...prev, ...evts] : evts));
        setHasMore(p < Math.ceil(data.total / data.per_page));
      } catch {
        // silent
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [customerId],
  );

  useEffect(() => {
    fetchEvents(1, false);
  }, [fetchEvents]);

  function loadMore() {
    const nextPage = page + 1;
    setPage(nextPage);
    fetchEvents(nextPage, true);
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
      </div>
    );
  }

  if (events.length === 0) {
    return (
      <EmptyState
        title="No events recorded yet"
        description="Customer events will appear here once activity is tracked."
      />
    );
  }

  return (
    <div className="relative">
      {/* Timeline line */}
      <div className="absolute bottom-0 left-4 top-0 w-0.5 bg-gray-200 dark:bg-gray-700" />

      <div className="space-y-4">
        {events.map((event) => {
          const config = eventIcons[event.type] ?? {
            icon: Circle,
            color: "text-gray-400",
          };
          const Icon = config.icon;
          const expanded = expandedId === event.id;

          return (
            <div key={event.id} className="relative ml-10">
              {/* Dot */}
              <div
                className={`absolute -left-10 top-1 flex h-8 w-8 items-center justify-center rounded-full bg-white dark:bg-gray-900 ${config.color}`}
              >
                <Icon className="h-4 w-4" />
              </div>

              {/* Content */}
              <button
                onClick={() => setExpandedId(expanded ? null : event.id)}
                className="w-full rounded-lg border border-gray-200 bg-white p-4 text-left transition-colors hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:hover:bg-gray-800"
              >
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    {event.title || event.type}
                  </span>
                  <span className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                    <span className="rounded-full bg-gray-100 px-2 py-0.5 dark:bg-gray-800">
                      {event.source}
                    </span>
                    {relativeTime(event.timestamp)}
                  </span>
                </div>
                {expanded && event.data && (
                  <pre className="mt-3 overflow-x-auto rounded-md bg-gray-50 p-3 text-xs text-gray-700 dark:bg-gray-800 dark:text-gray-300">
                    {JSON.stringify(event.data, null, 2)}
                  </pre>
                )}
              </button>
            </div>
          );
        })}
      </div>

      {hasMore && (
        <div className="ml-10 mt-4">
          <button
            onClick={loadMore}
            disabled={loadingMore}
            className="text-sm font-medium text-indigo-600 hover:underline disabled:opacity-50 dark:text-indigo-400"
          >
            {loadingMore ? "Loading..." : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}
