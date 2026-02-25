import { Link } from "react-router-dom";
import SeoMeta from "@/components/SeoMeta";
import {
  type SeoFamily,
  getHubByFamily,
  getSeoPagesByFamily,
  seoFamilyOrder,
  seoHubs,
  toSeoTitle,
} from "@/lib/seoCatalog";

interface SeoHubPageProps {
  family: SeoFamily;
}

export default function SeoHubPage({ family }: SeoHubPageProps) {
  const hub = getHubByFamily(family);
  const pages = getSeoPagesByFamily(family);

  const structuredData = {
    "@context": "https://schema.org",
    "@type": "CollectionPage",
    name: hub.title,
    description: hub.description,
    url: `https://pulsescore.app${hub.path}`,
    hasPart: pages.map((page) => ({
      "@type": "WebPage",
      name: page.h1,
      url: `https://pulsescore.app${page.path}`,
    })),
  };

  const relatedHubs = seoFamilyOrder
    .map((key) => seoHubs.find((hubEntry) => hubEntry.family === key))
    .filter((hubEntry): hubEntry is NonNullable<typeof hubEntry> => {
      if (!hubEntry) {
        return false;
      }
      return hubEntry.family !== family;
    })
    .slice(0, 4);

  return (
    <div className="min-h-screen bg-white px-6 py-12 text-gray-900 dark:bg-gray-950 dark:text-gray-100 sm:px-10 lg:px-14">
      <SeoMeta
        title={toSeoTitle(hub.title)}
        description={hub.description}
        path={hub.path}
        keywords={[
          `pulsescore ${hub.label.toLowerCase()}`,
          "customer health scoring",
          "b2b saas retention",
        ]}
        structuredData={structuredData}
      />

      <main className="mx-auto max-w-7xl">
        <header className="rounded-2xl border border-indigo-200 bg-indigo-50/70 p-8 dark:border-indigo-900 dark:bg-indigo-950/30">
          <p className="text-xs font-semibold tracking-[0.14em] text-indigo-600 uppercase dark:text-indigo-300">
            {hub.label}
          </p>
          <h1 className="mt-2 text-3xl font-extrabold tracking-tight sm:text-4xl">
            {hub.title}
          </h1>
          <p className="mt-3 max-w-3xl text-sm leading-6 text-gray-700 dark:text-gray-300">
            {hub.hero} {hub.description}
          </p>
          <div className="mt-5 flex flex-wrap gap-2 text-xs text-gray-600 dark:text-gray-300">
            <span className="rounded-full border border-gray-300 px-3 py-1 dark:border-gray-700">
              {pages.length} pages in this cluster
            </span>
            <span className="rounded-full border border-gray-300 px-3 py-1 dark:border-gray-700">
              Intent-driven templates
            </span>
            <span className="rounded-full border border-gray-300 px-3 py-1 dark:border-gray-700">
              Internal-link ready
            </span>
          </div>
        </header>

        <section className="mt-10 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {pages.map((page) => (
            <article
              key={page.path}
              className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-gray-800 dark:bg-gray-900"
            >
              <p className="text-xs font-semibold text-indigo-600 dark:text-indigo-300">
                {page.intent}
              </p>
              <h2 className="mt-1 text-lg font-semibold text-gray-900 dark:text-gray-100">
                <Link to={page.path} className="hover:text-indigo-600 dark:hover:text-indigo-300">
                  {page.h1}
                </Link>
              </h2>
              <p className="mt-2 text-sm text-gray-600 dark:text-gray-300">
                {page.description}
              </p>
              <p className="mt-3 text-xs text-gray-500 dark:text-gray-400">
                Keyword pattern: {page.keyword}
              </p>
            </article>
          ))}
        </section>

        <section className="mt-14 rounded-2xl border border-gray-200 bg-gray-50 p-6 dark:border-gray-800 dark:bg-gray-900">
          <h3 className="text-lg font-semibold">Explore related playbooks</h3>
          <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
            {relatedHubs.map((relatedHub) => (
              <Link
                key={relatedHub.path}
                to={relatedHub.path}
                className="rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm hover:border-indigo-300 hover:text-indigo-700 dark:border-gray-700 dark:bg-gray-950 dark:hover:border-indigo-700 dark:hover:text-indigo-300"
              >
                <p className="font-semibold">{relatedHub.label}</p>
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {relatedHub.description}
                </p>
              </Link>
            ))}
          </div>
          <div className="mt-6 flex flex-wrap gap-3">
            <Link
              to="/register"
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
            >
              Start free
            </Link>
            <Link
              to="/pricing"
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-semibold hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"
            >
              View pricing
            </Link>
          </div>
        </section>
      </main>
    </div>
  );
}
