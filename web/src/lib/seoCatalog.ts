import rawSeoPages from "@/content/seo-pages.json";
import rawFamilyConfig from "@/content/seo-family-config.json";

export type SeoFamily =
  | "templates"
  | "integrations"
  | "personas"
  | "comparisons"
  | "glossary"
  | "examples"
  | "curation";

export type SeoIntent = "informational" | "commercial" | "transactional";

export interface SeoPageSeed {
  family: SeoFamily;
  slug: string;
  entity: string;
  keyword: string;
  intent: SeoIntent;
}

export interface SeoPage extends SeoPageSeed {
  path: string;
  title: string;
  description: string;
  h1: string;
}

export interface SeoHub {
  family: SeoFamily;
  path: string;
  label: string;
  title: string;
  description: string;
  hero: string;
}

interface FamilyConfig {
  path: string;
  label: string;
  hubTitle: string;
  hubDescription: string;
  hero: string;
  titleTemplate: string;
  descriptionTemplate: string;
  h1Template: string;
}

const familyConfig = rawFamilyConfig as Record<SeoFamily, FamilyConfig>;

const seoPageSeeds = rawSeoPages as SeoPageSeed[];

function buildPath(seed: SeoPageSeed): string {
  return `${familyConfig[seed.family].path}/${seed.slug}`;
}

function renderTemplate(template: string, seed: SeoPageSeed): string {
  return template
    .replaceAll("{entity}", seed.entity)
    .replaceAll("{entityLower}", seed.entity.toLowerCase());
}

function withBrandTitle(base: string): string {
  const suffix = " | PulseScore";
  const maxLength = 60;
  const full = `${base}${suffix}`;

  if (full.length <= maxLength) {
    return full;
  }

  const available = maxLength - suffix.length;
  const truncatedBase =
    available > 1
      ? `${base.slice(0, available - 1).trimEnd()}…`
      : base.slice(0, available);

  return `${truncatedBase}${suffix}`;
}

export function toSeoTitle(base: string): string {
  return withBrandTitle(base);
}

function buildTitle(seed: SeoPageSeed): string {
  return withBrandTitle(renderTemplate(familyConfig[seed.family].titleTemplate, seed));
}

function buildDescription(seed: SeoPageSeed): string {
  return renderTemplate(familyConfig[seed.family].descriptionTemplate, seed);
}

function buildH1(seed: SeoPageSeed): string {
  return renderTemplate(familyConfig[seed.family].h1Template, seed);
}

export const seoHubs: SeoHub[] = (
  Object.keys(familyConfig) as SeoFamily[]
).map((family) => ({
  family,
  path: familyConfig[family].path,
  label: familyConfig[family].label,
  title: familyConfig[family].hubTitle,
  description: familyConfig[family].hubDescription,
  hero: familyConfig[family].hero,
}));

export const seoPages: SeoPage[] = seoPageSeeds.map((seed) => ({
  ...seed,
  path: buildPath(seed),
  title: buildTitle(seed),
  description: buildDescription(seed),
  h1: buildH1(seed),
}));

export function getHubByFamily(family: SeoFamily): SeoHub {
  const hub = seoHubs.find((item) => item.family === family);
  if (!hub) {
    throw new Error(`Unknown SEO family: ${family}`);
  }
  return hub;
}

export function getSeoPagesByFamily(family: SeoFamily): SeoPage[] {
  return seoPages.filter((page) => page.family === family);
}

export function getSeoPageByFamilyAndSlug(
  family: SeoFamily,
  slug: string,
): SeoPage | null {
  return (
    seoPages.find((page) => page.family === family && page.slug === slug) ??
    null
  );
}

export function getRelatedPages(page: SeoPage, limit = 6): SeoPage[] {
  const sameFamily = seoPages.filter(
    (item) => item.family === page.family && item.slug !== page.slug,
  );

  const crossFamily = seoPages.filter(
    (item) => item.family !== page.family && item.intent === page.intent,
  );

  const safeLimit = Math.max(0, limit);
  const sameFamilyLimit = Math.max(0, safeLimit - 2);

  return [...sameFamily.slice(0, sameFamilyLimit), ...crossFamily.slice(0, 2)].slice(
    0,
    safeLimit,
  );
}

export function getCrossFamilyHubs(activeFamily: SeoFamily): SeoHub[] {
  return seoHubs.filter((hub) => hub.family !== activeFamily).slice(0, 4);
}

export const seoFamilyOrder: SeoFamily[] = [
  "templates",
  "integrations",
  "personas",
  "comparisons",
  "examples",
  "curation",
  "glossary",
];
