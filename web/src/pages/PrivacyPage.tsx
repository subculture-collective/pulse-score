import { Link } from "react-router-dom";
import SeoMeta from "@/components/SeoMeta";

export default function PrivacyPage() {
  return (
    <main className="min-h-screen bg-gray-50 px-6 py-12 dark:bg-gray-950 sm:px-10 lg:px-14">
      <SeoMeta
        title="Privacy Policy | PulseScore"
        description="Read the PulseScore privacy policy and data handling practices."
        path="/privacy"
        noIndex
      />

      <div className="mx-auto max-w-3xl rounded-2xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-800 dark:bg-gray-900">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Privacy Policy</h1>
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          Last updated: February 24, 2026
        </p>

        <div className="mt-6 space-y-5 text-sm leading-7 text-gray-700 dark:text-gray-300">
          <p>
            This is placeholder policy content for MVP use. Replace with legal-reviewed
            copy before production launch.
          </p>
          <p>
            PulseScore collects account profile information and integration data only
            for product functionality, analytics, and support.
          </p>
          <p id="cookies">
            We use essential cookies for authentication and optional analytics cookies
            for product improvement.
          </p>
        </div>

        <Link
          to="/"
          className="mt-8 inline-flex rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
        >
          Back to home
        </Link>
      </div>
    </main>
  );
}
