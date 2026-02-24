import { useEffect } from "react";

type StructuredData = Record<string, unknown> | Record<string, unknown>[];

interface SeoMetaProps {
  title: string;
  description: string;
  path: string;
  imagePath?: string;
  keywords?: string[];
  type?: "website" | "article";
  noIndex?: boolean;
  structuredData?: StructuredData;
}

function upsertMeta(
  attribute: "name" | "property",
  key: string,
  content: string,
) {
  let meta = document.head.querySelector(
    `meta[${attribute}="${key}"]`,
  ) as HTMLMetaElement | null;

  if (!meta) {
    meta = document.createElement("meta");
    meta.setAttribute(attribute, key);
    document.head.appendChild(meta);
  }

  meta.setAttribute("content", content);
}

export default function SeoMeta({
  title,
  description,
  path,
  imagePath = "/og-card.svg",
  keywords,
  type = "website",
  noIndex = false,
  structuredData,
}: SeoMetaProps) {
  useEffect(() => {
    const origin = window.location.origin;
    const url = new URL(path, origin).toString();
    const image = imagePath.startsWith("http")
      ? imagePath
      : new URL(imagePath, origin).toString();

    document.title = title;

    upsertMeta("name", "description", description);
    if (keywords?.length) {
      upsertMeta("name", "keywords", keywords.join(", "));
    }

    const robotsValue = noIndex ? "noindex, nofollow" : "index, follow";
    upsertMeta("name", "robots", robotsValue);

    upsertMeta("property", "og:title", title);
    upsertMeta("property", "og:description", description);
    upsertMeta("property", "og:type", type);
    upsertMeta("property", "og:url", url);
    upsertMeta("property", "og:image", image);

    upsertMeta("name", "twitter:card", "summary_large_image");
    upsertMeta("name", "twitter:title", title);
    upsertMeta("name", "twitter:description", description);
    upsertMeta("name", "twitter:image", image);

    let canonical = document.head.querySelector(
      'link[rel="canonical"]',
    ) as HTMLLinkElement | null;
    if (!canonical) {
      canonical = document.createElement("link");
      canonical.setAttribute("rel", "canonical");
      document.head.appendChild(canonical);
    }
    canonical.href = url;

    const structuredDataId = "app-structured-data";
    const existingScript = document.getElementById(structuredDataId);
    if (existingScript) {
      existingScript.remove();
    }

    if (structuredData) {
      const script = document.createElement("script");
      script.id = structuredDataId;
      script.type = "application/ld+json";
      script.text = JSON.stringify(structuredData);
      document.head.appendChild(script);
    }
  }, [
    description,
    imagePath,
    keywords,
    noIndex,
    path,
    structuredData,
    title,
    type,
  ]);

  return null;
}
