import { Link } from "react-router-dom";
import SeoMeta from "@/components/SeoMeta";
import HeroSection from "@/components/landing/HeroSection";
import FeaturesSection from "@/components/landing/FeaturesSection";
import PricingSection from "@/components/landing/PricingSection";
import SocialProofSection from "@/components/landing/SocialProofSection";
import FooterSection from "@/components/landing/FooterSection";

const landingStructuredData = [
  {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: "PulseScore",
    url: "https://pulsescore.app",
    logo: "https://pulsescore.app/og-card.svg",
    sameAs: ["https://github.com/subculture-collective/pulse-score"],
  },
  {
    "@context": "https://schema.org",
    "@type": "WebApplication",
    name: "PulseScore",
    applicationCategory: "BusinessApplication",
    operatingSystem: "Web",
    offers: [
      { "@type": "Offer", name: "Free", price: "0", priceCurrency: "USD" },
      { "@type": "Offer", name: "Growth", price: "49", priceCurrency: "USD" },
      { "@type": "Offer", name: "Scale", price: "149", priceCurrency: "USD" },
    ],
    description:
      "PulseScore helps B2B SaaS teams detect churn risk early with customer health scoring.",
  },
];

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-white text-gray-900 dark:bg-gray-950 dark:text-gray-100">
      <SeoMeta
        title="PulseScore | Know Customer Health Before They Churn"
        description="Connect Stripe, HubSpot, and Intercom to monitor customer health and reduce churn with proactive alerts."
        keywords={[
          "customer health score",
          "churn prevention",
          "b2b saas",
          "stripe analytics",
          "customer success software",
        ]}
        path="/"
        type="website"
        structuredData={landingStructuredData}
      />

      <header className="sticky top-0 z-20 border-b border-gray-200/80 bg-white/95 backdrop-blur dark:border-gray-800 dark:bg-gray-950/90">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-3 sm:px-10 lg:px-14">
          <Link
            to="/"
            className="text-lg font-bold text-indigo-600 dark:text-indigo-300"
          >
            PulseScore
          </Link>

          <nav className="hidden items-center gap-6 text-sm text-gray-600 md:flex dark:text-gray-300">
            <a
              href="#features"
              className="hover:text-indigo-600 dark:hover:text-indigo-300"
            >
              Features
            </a>
            <a
              href="#pricing"
              className="hover:text-indigo-600 dark:hover:text-indigo-300"
            >
              Pricing
            </a>
          </nav>

          <div className="flex items-center gap-2">
            <Link
              to="/login"
              className="rounded-lg px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-800"
            >
              Sign in
            </Link>
            <Link
              to="/register"
              className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
            >
              Start free
            </Link>
          </div>
        </div>
      </header>

      <HeroSection />
      <FeaturesSection />
      <PricingSection />
      <SocialProofSection />
      <FooterSection />
    </div>
  );
}
