interface IntegrationStatusBadgeProps {
  status: "connected" | "syncing" | "error" | "disconnected";
}

const statusStyles = {
  connected: {
    dot: "bg-green-500",
    text: "text-green-700 dark:text-green-300",
    label: "Connected",
  },
  syncing: {
    dot: "bg-blue-500 animate-pulse",
    text: "text-blue-700 dark:text-blue-300",
    label: "Syncing",
  },
  error: {
    dot: "bg-red-500",
    text: "text-red-700 dark:text-red-300",
    label: "Error",
  },
  disconnected: {
    dot: "bg-gray-400 dark:bg-gray-500",
    text: "text-gray-500 dark:text-gray-400",
    label: "Disconnected",
  },
};

export default function IntegrationStatusBadge({
  status,
}: IntegrationStatusBadgeProps) {
  const s = statusStyles[status] ?? statusStyles.disconnected;

  return (
    <span className={`inline-flex items-center gap-1.5 text-sm ${s.text}`}>
      <span className={`h-2 w-2 rounded-full ${s.dot}`} />
      {s.label}
    </span>
  );
}
