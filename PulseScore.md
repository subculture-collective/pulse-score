SMB Customer Health Scoring Platform
Market map
Primary segment: B2B SaaS companies with 50–500 paying customers, 10–100 employees, $500K–$10M ARR. These companies have enough customers that spreadsheet tracking breaks down but not enough revenue to justify $12K+/yr CS platforms. Estimated 5,000–15,000 companies globally in this segment, with approximately 10,000 in the US alone (based on ~17,000 US SaaS companies, majority being early/growth stage).
Adjacent markets for expansion:

Subscription businesses (membership sites, subscription boxes, media): Same health-scoring need, different data sources
Professional services firms with retainer clients: Accounting firms, agencies, MSPs tracking client health
SaaS companies with 500–2,000 customers upgrading from lightweight tool (natural upsell path to mid-market)

Buyer journey: SaaS founder realizes churn is rising → Googles "customer health score template" → Downloads spreadsheet template → Realizes manual data pulls are unsustainable → Searches for affordable CS tool → Finds everything costs $10K+/yr → Settles back into spreadsheets OR finds PulseScore at $49/mo → Connects Stripe → Sees health scores in 15 minutes → Invites CS team → Expands usage.
Market size estimates:

Customer Success Platforms market: $1.86B in 2024, projected $9.17B by 2032 at 22.1% CAGR (industry data via Custify)
SME segment: 39.4% of B2B SaaS market, growing at 22.8% CAGR
Serviceable addressable market: 10,000 target SaaS companies × $1,200/yr average = $12M/yr
Broader TAM including all subscription businesses: $36M–$60M/yr
Realistic capture (3 years): 500–1,000 customers = $600K–$1.2M ARR

Unit economics rough cut
Customer acquisition cost by channel:
ChannelEstimated CACVolumeNotesSEO/content$20–$50Medium (10–20/mo at maturity)Health score templates, comparison pages, blog. 6–12 month ramp.Stripe App Marketplace$10–$30Medium (5–15/mo)Built-in distribution to SaaS companies using StripeHubSpot Marketplace$15–$40Low-medium (3–10/mo)Second marketplace listingCommunity/social$15–$30Low (5–10/mo)Reddit, Indie Hackers, Twitter; labor-intensiveCold outreach$50–$100Low (3–5/mo)LinkedIn DMs to SaaS CS leads; higher touchProduct Hunt launch~$5Spike (50–100 signups, one-time)Single launch event; not repeatable
Blended CAC target: $30–$60
LTV at $80/mo ARPU, 18-month average lifetime: $1,440
LTV:CAC ratio: 24:1 to 48:1 (excellent)
Pricing model: Tiered SaaS subscription. Free tier drives awareness and marketplace installs. Paid conversion at 5–10% of free users (benchmarked against similar PLG tools like Tally's conversion rates).
Margin considerations:

Infrastructure: PostgreSQL on managed DB (~$50/mo), Go API server (~$20/mo), React frontend on Vercel (~$20/mo). Total infra: ~$100/mo at early scale
API costs: Stripe API (free), HubSpot API (free for most calls), Intercom API (free for basic access)
LLM costs (if using AI for insights): ~$0.01–$0.05 per customer health analysis with GPT-4o-mini
Gross margin: 85–90% at scale

Build plan
Architecture sketch:
[Stripe API] ──→ [Go Ingestion Service] ──→ [PostgreSQL]
[HubSpot API] ──→ [Go Ingestion Service] ──→      ↓
[Intercom API] ──→ [Go Ingestion Service] ──→ [Health Scoring Engine (Go)]
                                                    ↓
                                            [REST API (Go)]
                                                    ↓
                                            [React Dashboard]
                                                    ↓
                                            [Email Alerts (SendGrid)]
Tech stack: React/TypeScript frontend, Go API + background workers, PostgreSQL, Stripe Connect for billing, SendGrid for alerts. Hosted on Railway or Render initially.
Week-by-week MVP build plan:
WeekDeliverableDetails1Stripe integration + data modelOAuth flow, subscription/payment data sync, PostgreSQL schema for customers, events, health scores2Health scoring engine + APIWeighted scoring algorithm (payment recency, MRR trend, failed payment history, support ticket volume), REST API endpoints3React dashboardCustomer list with health scores, color-coded status (green/yellow/red), sorting/filtering, individual customer detail page with timeline4HubSpot integration + alertsHubSpot OAuth, contact/deal data enrichment, email alerts when health score drops below threshold5Intercom integration + onboardingIntercom ticket data sync, self-serve onboarding wizard, Stripe billing integration for paid plans6Polish, docs, launch prepLanding page, documentation, Stripe App Marketplace submission, Product Hunt prep, beta user onboarding
Key integrations/APIs:

Stripe Connect (OAuth, Subscriptions API, Events API, Invoices API)
HubSpot API (Contacts, Deals, Companies — free tier API access)
Intercom API (Conversations, Contacts)
SendGrid (transactional alerts)
PostHog/Mixpanel (future: product usage data via customer's analytics)

Moat strategy: wedge → expansion → defensibility
Wedge (months 1–6): "The $49/mo Stripe customer health dashboard." Win on price, speed of setup (15 minutes vs. weeks), and Stripe-first positioning. Build initial customer base of 50–200 SaaS companies through marketplaces and SEO.
Expansion (months 6–18): Add automated playbooks (trigger email when health drops), team collaboration features (assign at-risk accounts), more data source integrations (Zendesk, Salesforce, product analytics tools). Raise ARPU to $100–$200/mo with Growth/Scale tiers.
Defensibility (months 18+): As customers accumulate historical health data, switching costs increase (can't easily export 18 months of trend data and scoring history). Network effects emerge through anonymized benchmarking ("Your health score distribution vs. SaaS companies your size"). Expand into adjacent verticals (subscription boxes, agencies, membership businesses). Build an integration marketplace where customers contribute data source connectors.
Long-term moat: Data gravity. The more customer health data PulseScore accumulates across its customer base, the better its benchmarking, predictive models, and scoring algorithms become. This is a compounding advantage competitors can't replicate without equivalent data scale.
"Why this wins" narrative

The pricing gap is absurd. There is literally a 100x price difference between spreadsheets ($0) and the cheapest mid-market CS platform (~$10K/yr). PulseScore fills this gap at $49–$99/mo with a product that's better than spreadsheets and more accessible than enterprise tools.
Stripe-first is a distribution cheat code. Every B2B SaaS company in the target segment uses Stripe. The Stripe App Marketplace provides built-in distribution to exactly the right buyer. No other CS platform has Stripe marketplace presence as a primary distribution channel.
The buyer is the builder. SaaS founders and CS leads are technical enough to self-serve through an onboarding wizard but time-strapped enough to value "connect Stripe, get health scores in 15 minutes." This enables product-led growth without needing a sales team.
CS platform layoffs create demand for automation. With 44.2% of companies laying off CSMs, the teams that remain need tools that scale their impact without scaling headcount. A $49/mo tool that replaces a spreadsheet operated by a $120K/yr CSM has obvious ROI.
The tech stack is a perfect fit. React dashboard + Go API + PostgreSQL is exactly the founder's stack. Stripe and HubSpot APIs are well-documented with Go SDKs. No new technologies to learn. The entire MVP is within the founder's existing capability set.
