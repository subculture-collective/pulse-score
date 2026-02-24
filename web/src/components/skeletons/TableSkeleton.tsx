export default function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
      {/* Header */}
      <div className="flex gap-4 border-b border-gray-200 bg-gray-50 px-6 py-3 dark:border-gray-700 dark:bg-gray-800">
        {[...Array(5)].map((_, i) => (
          <div
            key={i}
            className="h-4 flex-1 animate-pulse rounded bg-gray-200 dark:bg-gray-700"
          />
        ))}
      </div>
      {/* Rows */}
      {[...Array(rows)].map((_, row) => (
        <div
          key={row}
          className="flex gap-4 border-b border-gray-100 px-6 py-4 last:border-0 dark:border-gray-800"
        >
          {[...Array(5)].map((_, col) => (
            <div
              key={col}
              className="h-4 flex-1 animate-pulse rounded bg-gray-200 dark:bg-gray-700"
            />
          ))}
        </div>
      ))}
    </div>
  );
}
