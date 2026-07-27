import { ArrowLeft, ExternalLink, Terminal } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import {
  SITE_NAME,
  SITE_URL,
  SOCIAL_IMAGE,
} from "../site";
import ThemeToggle from "../theme-toggle";

const ABOUT_DESCRIPTION =
  "How wthrtxt.com turns NOAA, ACIS and Open-Meteo data into a fast local weather forecast for browsers and curl.";

export const dynamic = "force-static";

export const metadata: Metadata = {
  alternates: {
    canonical: "/about/",
  },
  description: ABOUT_DESCRIPTION,
  openGraph: {
    description: ABOUT_DESCRIPTION,
    images: [SOCIAL_IMAGE],
    siteName: SITE_NAME,
    title: "About wthrtxt.com",
    type: "website",
    url: "/about/",
  },
  title: "About",
  twitter: {
    card: "summary_large_image",
    description: ABOUT_DESCRIPTION,
    images: [{ alt: SOCIAL_IMAGE.alt, url: SOCIAL_IMAGE.url }],
    title: "About wthrtxt.com",
  },
};

const aboutJsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@id": `${SITE_URL}/about/#about`,
      "@type": "AboutPage",
      about: {
        "@id": `${SITE_URL}/#weather-app`,
      },
      description: ABOUT_DESCRIPTION,
      inLanguage: "en-US",
      isPartOf: {
        "@id": `${SITE_URL}/#website`,
      },
      name: "About wthrtxt.com",
      primaryImageOfPage: {
        "@type": "ImageObject",
        contentUrl: `${SITE_URL}${SOCIAL_IMAGE.url}`,
        height: SOCIAL_IMAGE.height,
        width: SOCIAL_IMAGE.width,
      },
      url: `${SITE_URL}/about/`,
    },
    {
      "@type": "BreadcrumbList",
      itemListElement: [
        {
          "@type": "ListItem",
          item: `${SITE_URL}/`,
          name: "Weather",
          position: 1,
        },
        {
          "@type": "ListItem",
          item: `${SITE_URL}/about/`,
          name: "About",
          position: 2,
        },
      ],
    },
  ],
};

function SourceLink({
  children,
  href,
}: {
  children: React.ReactNode;
  href: string;
}) {
  return (
    <a href={href} rel="noreferrer">
      {children}
      <ExternalLink aria-hidden="true" size={11} />
    </a>
  );
}

export default function AboutPage() {
  return (
    <main className="weather-shell">
      <script
        dangerouslySetInnerHTML={{
          __html: JSON.stringify(aboutJsonLd).replace(/</g, "\\u003c"),
        }}
        id="about-json-ld"
        type="application/ld+json"
      />

      <header className="site-header">
        <div className="header-line">
          <Link className="site-title" href="/">
            wthrtxt.com
          </Link>
          <div className="header-actions">
            <Link
              aria-label="Back to weather"
              className="icon-button"
              href="/"
            >
              <ArrowLeft aria-hidden="true" size={14} />
            </Link>
            <ThemeToggle />
          </div>
        </div>
      </header>

      <div aria-hidden="true" className="ascii-rule">
        ============================================================
      </div>

      <article className="about-page">
        <p className="section-label">about</p>
        <h1>Local weather, minus the weather-site clutter.</h1>
        <p className="about-lede">
          wthrtxt.com is a small, free weather reader for current conditions,
          hour-by-hour details, seven-day forecasts, rainfall, and daily
          temperature records.
        </p>

        <section>
          <h2>How it works</h2>
          <p>
            Use your browser location or search for a place. Forecasts are
            cached briefly for speed, and every location gets a readable,
            shareable URL. Browser requests go to the weather sources directly;
            the Go server renders the same forecast as plain text for terminals.
          </p>
        </section>

        <section>
          <h2>Sources</h2>
          <ul className="source-list">
            <li>
              <SourceLink href="https://www.weather.gov/documentation/services-web-api">
                NOAA / National Weather Service
              </SourceLink>
              <span>U.S. forecasts and station observations</span>
            </li>
            <li>
              <SourceLink href="https://docs.rcc-acis.org/acisws/">
                ACIS
              </SourceLink>
              <span>U.S. station climate records</span>
            </li>
            <li>
              <SourceLink href="https://open-meteo.com/en/docs">
                Open-Meteo
              </SourceLink>
              <span>global forecasts and ERA5-Land record estimates</span>
            </li>
            <li>
              <SourceLink href="https://github.com/komoot/photon">
                OpenStreetMap Photon
              </SourceLink>
              <span>live place search</span>
            </li>
            <li>
              <SourceLink href="https://nominatim.org/release-docs/latest/api/Overview/">
                OpenStreetMap Nominatim
              </SourceLink>
              <span>location URL and coordinate lookup</span>
            </li>
            <li>
              <SourceLink href="https://dev.maxmind.com/geoip/geolite2-free-geolocation-data/">
                MaxMind GeoLite2
              </SourceLink>
              <span>approximate location for terminal requests</span>
            </li>
          </ul>
          <p className="about-note">
            Open-Meteo past-hour values and ERA5-Land records are model-based
            estimates, not official station observations.
          </p>
        </section>

        <section>
          <h2>
            <Terminal aria-hidden="true" size={14} />
            Browser or curl
          </h2>
          <p>Open any location normally, or ask the same URL for plain text:</p>
          <pre>
            <code>{`curl https://wthrtxt.com
curl https://wthrtxt.com/seattle
curl 'https://wthrtxt.com/seattle?format=text'`}</code>
          </pre>
        </section>
      </article>
    </main>
  );
}
