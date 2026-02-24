import {
  Activity,
  BellRing,
  ChartNoAxesCombined,
  Compass,
  Plug,
} from "lucide-react";

const features = [
  {
    title: "Health scoring that updates itself",
    description:
      "Automatically score every customer using product usage, billing signals, and engagement trends.",
    icon: Activity,
  },
  {
    title: "Integration hub in one place",
    description:
      "Connect Stripe, HubSpot, and Intercom without duct-tape scripts or spreadsheets.",
    icon: Plug,
  },
  {
    title: "Early warning alerts",
    description:
      "Get notified before renewal risk escalates so your team can intervene at the right moment.",
    icon: BellRing,
  },
  {
    title: "Customer insight timeline",
    description:
      "See score movement, events, and account context in one clear story for every customer.",
    icon: Compass,
  },
  {
    title: "Analytics that explain the why",
    description:
      "Track portfolio health distribution and identify the factors driving churn risk over time.",
    icon: ChartNoAxesCombined,
  },
];

export default function FeaturesSection() {
  return (
    <section
      id="features"
      className="bg-gray-50 px-6 py-16 dark:bg-gray-900 sm:px-10 lg:px-14 lg:py-24"
    >
      <div className="mx-auto max-w-7xl">
        <div className="max-w-3xl">
          <h2 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100 sm:text-4xl">
            Built for lean CS teams that still need enterprise-grade signal.
          </h2>
          <p className="mt-3 text-gray-600 dark:text-gray-300">
            Focus effort where it matters most, with proactive visibility
            instead of reactive firefighting.
          </p>
        </div>

        <div className="mt-10 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {features.map((feature) => {
            const Icon = feature.icon;
            return (
              <article
                key={feature.title}
                className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-gray-800 dark:bg-gray-950"
              >
                <div className="mb-4 inline-flex rounded-lg bg-indigo-100 p-2 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300">
                  <Icon className="h-5 w-5" />
                </div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {feature.title}
                </h3>
                <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">
                  {feature.description}
                </p>
              </article>
            );
          })}
        </div>
      </div>
    </section>
  );
}
