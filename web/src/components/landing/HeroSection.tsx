import { ArrowRight, BarChart3, ShieldCheck, Zap } from "lucide-react";
import { Link } from "react-router-dom";

export default function HeroSection() {
  return (
    <section className="relative overflow-hidden bg-white px-6 pb-16 pt-14 dark:bg-gray-950 sm:px-10 lg:px-14 lg:pb-24 lg:pt-20">
      <div className="pointer-events-none absolute -left-24 top-0 h-80 w-80 rounded-full bg-indigo-100 blur-3xl dark:bg-indigo-900/40" />
      <div className="pointer-events-none absolute -right-28 bottom-0 h-80 w-80 rounded-full bg-cyan-100 blur-3xl dark:bg-cyan-900/30" />

      <div className="relative mx-auto grid w-full max-w-7xl gap-12 lg:grid-cols-[1.1fr_1fr] lg:items-center">
        <div>
          <span className="inline-flex items-center rounded-full border border-indigo-200 bg-indigo-50 px-3 py-1 text-xs font-semibold tracking-wide text-indigo-700 uppercase dark:border-indigo-800 dark:bg-indigo-950/60 dark:text-indigo-300">
            Customer health intelligence
          </span>
          <h1 className="mt-5 text-4xl font-extrabold tracking-tight text-gray-900 dark:text-gray-100 sm:text-5xl lg:text-6xl">
            Know customer risk before churn hits your MRR.
          </h1>
          <p className="mt-5 max-w-2xl text-lg text-gray-600 dark:text-gray-300">
            PulseScore connects Stripe, HubSpot, and Intercom to surface at-risk
            accounts in minutes — so your team can act early with confidence.
          </p>

          <div className="mt-8 flex flex-wrap gap-3">
            <Link
              to="/register"
              className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-3 text-sm font-semibold text-white shadow-sm transition hover:bg-indigo-700"
            >
              Start free
              <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              to="/login"
              className="inline-flex items-center rounded-xl border border-gray-300 bg-white px-5 py-3 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
            >
              Sign in
            </Link>
          </div>

          <div className="mt-8 grid max-w-xl grid-cols-1 gap-3 text-sm text-gray-600 dark:text-gray-300 sm:grid-cols-3">
            <div className="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-800 dark:bg-gray-900/70">
              <Zap className="mb-1 h-4 w-4 text-indigo-500" />
              15-minute setup
            </div>
            <div className="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-800 dark:bg-gray-900/70">
              <BarChart3 className="mb-1 h-4 w-4 text-indigo-500" />
              Real-time scoring
            </div>
            <div className="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-800 dark:bg-gray-900/70">
              <ShieldCheck className="mb-1 h-4 w-4 text-indigo-500" />
              Privacy-first
            </div>
          </div>
        </div>

        <div className="relative">
          <div className="rounded-2xl border border-indigo-200 bg-white p-4 shadow-xl shadow-indigo-100/60 dark:border-indigo-900 dark:bg-gray-900 dark:shadow-none">
            <div className="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-950">
              <div className="mb-4 flex items-center justify-between">
                <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                  Health Score Snapshot
                </p>
                <span className="rounded-full bg-emerald-100 px-2 py-1 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                  Live
                </span>
              </div>

              <div className="space-y-3">
                {[
                  {
                    name: "Northwind SaaS",
                    score: 89,
                    tone: "from-emerald-400 to-emerald-600 dark:from-emerald-500 dark:to-emerald-400",
                  },
                  {
                    name: "Orbit Analytics",
                    score: 64,
                    tone: "from-amber-400 to-amber-600 dark:from-amber-500 dark:to-amber-400",
                  },
                  {
                    name: "Acme Ops",
                    score: 37,
                    tone: "from-rose-400 to-rose-600 dark:from-rose-500 dark:to-rose-400",
                  },
                ].map((customer) => (
                  <div
                    key={customer.name}
                    className="rounded-lg bg-white p-3 dark:bg-gray-900"
                  >
                    <div className="mb-2 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                      <span>{customer.name}</span>
                      <span className="font-semibold">
                        {customer.score}/100
                      </span>
                    </div>
                    <div className="h-2 rounded-full bg-gray-200 dark:bg-gray-800">
                      <div
                        className={`h-2 rounded-full bg-gradient-to-r ${customer.tone}`}
                        style={{ width: `${customer.score}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
