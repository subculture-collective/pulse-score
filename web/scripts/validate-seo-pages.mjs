import { readFile } from "node:fs/promises";
import { readFileSync } from "node:fs";

const FAMILY_CONFIG = JSON.parse(
  readFileSync(new URL("../src/content/seo-family-config.json", import.meta.url), "utf8"),
);

const FAMILY_PATH_PREFIX = Object.fromEntries(
  Object.entries(FAMILY_CONFIG).map(([family, config]) => [family, config.path]),
);

const VALID_FAMILIES = new Set(Object.keys(FAMILY_PATH_PREFIX));

const FAMILY_MINIMUMS = {
  templates: 20,
  integrations: 15,
  personas: 10,
  comparisons: 5,
  glossary: 20,
  examples: 10,
  curation: 10,
};

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function isSlugValid(slug) {
  return /^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug);
}

async function main() {
  const raw = await readFile(new URL("../src/content/seo-pages.json", import.meta.url), "utf8");
  const pages = JSON.parse(raw);
  const families = Object.keys(FAMILY_PATH_PREFIX);

  assert(Array.isArray(pages), "SEO catalog must be an array.");
  assert(pages.length >= 50, `Expected at least 50 SEO pages, found ${pages.length}.`);

  const counts = Object.fromEntries(families.map((family) => [family, 0]));

  const fullPaths = new Set();
  const keywords = new Set();

  for (const page of pages) {
    assert(typeof page === "object" && page !== null, "Each SEO entry must be an object.");

    const { family, slug, entity, keyword, intent } = page;

    assert(VALID_FAMILIES.has(family), `Invalid family: ${family}`);
    assert(isSlugValid(slug), `Invalid slug format: ${slug}`);
    assert(typeof entity === "string" && entity.trim().length >= 3, `Invalid entity for slug: ${slug}`);
    assert(
      typeof keyword === "string" && keyword.trim().split(/\s+/).length >= 2,
      `Keyword must have at least 2 words for slug: ${slug}`,
    );
    assert(
      intent === "informational" || intent === "commercial" || intent === "transactional",
      `Invalid intent for slug: ${slug}`,
    );

    const fullPath = `${FAMILY_PATH_PREFIX[family]}/${slug}`;
    assert(!fullPaths.has(fullPath), `Duplicate SEO path: ${fullPath}`);
    fullPaths.add(fullPath);

    const keywordKey = `${family}:${keyword.toLowerCase().trim()}`;
    assert(!keywords.has(keywordKey), `Duplicate keyword within family: ${keyword}`);
    keywords.add(keywordKey);

    counts[family] += 1;
  }

  for (const [family, minimum] of Object.entries(FAMILY_MINIMUMS)) {
    assert(
      counts[family] >= minimum,
      `Family ${family} has ${counts[family]} pages, expected at least ${minimum}.`,
    );
  }

  console.log("SEO catalog validation passed.");
  console.log("Page counts by family:", counts);
}

main().catch((error) => {
  console.error(`SEO catalog validation failed: ${error.message}`);
  process.exitCode = 1;
});
