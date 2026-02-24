export default function ChartSkeleton() {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
      <div className="mb-4 h-4 w-40 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
      <div className="h-64 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
    </div>
  );
}
