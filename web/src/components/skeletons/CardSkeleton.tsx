export default function CardSkeleton() {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
      <div className="mb-4 h-4 w-24 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
      <div className="mb-2 h-8 w-32 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
      <div className="h-4 w-20 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
    </div>
  );
}
