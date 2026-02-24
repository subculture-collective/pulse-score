import { Link } from "react-router-dom";
import { FileQuestion } from "lucide-react";

export default function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gray-50 p-8 text-center dark:bg-gray-950">
      <FileQuestion className="mb-6 h-16 w-16 text-gray-400 dark:text-gray-500" />
      <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
        Page not found
      </h1>
      <p className="mt-2 text-gray-500 dark:text-gray-400">
        The page you're looking for doesn't exist or has been moved.
      </p>
      <Link
        to="/"
        className="mt-6 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
      >
        Back to home
      </Link>
    </div>
  );
}
