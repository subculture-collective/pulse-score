import { Quote } from "lucide-react";

const logos = [
  "Acme Cloud",
  "Northwind",
  "Orbit Analytics",
  "Pixel Ops",
  "Summit AI",
  "Atlas CRM",
];

const testimonials = [
  {
    quote:
      "PulseScore helped us prioritize the exact 12 accounts that were quietly slipping. We recovered 4 before renewal.",
    name: "Jamie Patel",
    role: "Head of Customer Success",
    company: "Northwind SaaS",
  },
  {
    quote:
      "We moved from guessing risk in spreadsheets to seeing clear health movement every morning.",
    name: "Morgan Lee",
    role: "Founder",
    company: "Orbit Analytics",
  },
  {
    quote:
      "The Stripe + HubSpot sync made onboarding painless. Our team had actionable signals on day one.",
    name: "Riley Chen",
    role: "RevOps Lead",
    company: "Acme Cloud",
  },
];

const metrics = [
  { value: "500+", label: "customers monitored" },
  { value: "30%", label: "average churn reduction potential" },
  { value: "15 min", label: "time to first score" },
];

export default function SocialProofSection() {
  return (
    <section className="bg-gray-50 px-6 py-16 dark:bg-gray-900 sm:px-10 lg:px-14 lg:py-24">
      <div className="mx-auto max-w-7xl">
        <p className="text-xs font-semibold tracking-[0.14em] text-gray-500 uppercase dark:text-gray-400">
          Placeholder logos & testimonials for MVP (replace with production brand assets)
        </p>

        <div className="mt-5 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          {logos.map((logo) => (
            <div
              key={logo}
              className="rounded-lg border border-gray-200 bg-white px-3 py-2 text-center text-sm font-semibold text-gray-500 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300"
            >
              {logo}
            </div>
          ))}
        </div>

        <div className="mt-10 grid grid-cols-1 gap-4 lg:grid-cols-3">
          {metrics.map((metric) => (
            <div
              key={metric.label}
              className="rounded-2xl border border-indigo-200 bg-white p-6 text-center dark:border-indigo-900 dark:bg-gray-950"
            >
              <p className="text-3xl font-extrabold text-indigo-600 dark:text-indigo-300">{metric.value}</p>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-300">{metric.label}</p>
            </div>
          ))}
        </div>

        <div className="mt-10 grid grid-cols-1 gap-4 lg:grid-cols-3">
          {testimonials.map((testimonial) => (
            <figure
              key={`${testimonial.name}-${testimonial.company}`}
              className="rounded-2xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-950"
            >
              <Quote className="h-5 w-5 text-indigo-500" />
              <blockquote className="mt-3 text-sm leading-6 text-gray-700 dark:text-gray-200">
                “{testimonial.quote}”
              </blockquote>
              <figcaption className="mt-4 text-xs text-gray-500 dark:text-gray-400">
                <span className="font-semibold text-gray-700 dark:text-gray-200">{testimonial.name}</span>
                <span> · {testimonial.role}, {testimonial.company}</span>
              </figcaption>
            </figure>
          ))}
        </div>
      </div>
    </section>
  );
}
