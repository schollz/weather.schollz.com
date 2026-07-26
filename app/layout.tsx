import type { Metadata, Viewport } from "next";
import "./globals.css";

export const dynamic = "force-static";

const SITE_NAME = "weather.schollz.com";
const SITE_URL = "https://weather.schollz.com";
const DESCRIPTION =
  "Local worldwide weather with current conditions, hourly details, seven-day forecasts, and daily temperature records from NOAA, ACIS, and Open-Meteo.";

export const metadata: Metadata = {
  alternates: {
    canonical: "/",
  },
  applicationName: SITE_NAME,
  description: DESCRIPTION,
  keywords: [
    "weather",
    "local weather",
    "worldwide weather",
    "NOAA",
    "Open-Meteo",
    "hourly forecast",
    "seven-day forecast",
    "climate records",
    "ACIS",
  ],
  metadataBase: new URL(SITE_URL),
  openGraph: {
    description: DESCRIPTION,
    locale: "en_US",
    siteName: SITE_NAME,
    title: "Local worldwide weather",
    type: "website",
    url: "/",
  },
  robots: {
    follow: true,
    index: true,
  },
  title: `${SITE_NAME} — local worldwide weather`,
  twitter: {
    card: "summary",
    description: DESCRIPTION,
    title: "Local worldwide weather",
  },
};

export const viewport: Viewport = {
  colorScheme: "dark light",
  initialScale: 1,
  themeColor: [
    { color: "#0d0d0d", media: "(prefers-color-scheme: dark)" },
    { color: "#f3f2ec", media: "(prefers-color-scheme: light)" },
  ],
  width: "device-width",
};

const jsonLd = {
  "@context": "https://schema.org",
  "@type": "WebApplication",
  applicationCategory: "Weather",
  browserRequirements: "Requires JavaScript",
  description: DESCRIPTION,
  featureList: [
    "Current local weather",
    "Hourly observations, estimates, and forecasts",
    "Seven-day worldwide forecast",
    "NOAA and ACIS data in supported U.S. areas",
    "Open-Meteo forecasts and estimated records worldwide",
    "Shareable locations",
  ],
  isAccessibleForFree: true,
  name: SITE_NAME,
  operatingSystem: "Any",
  url: `${SITE_URL}/`,
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <link href="favicon.svg" rel="icon" type="image/svg+xml" />
        <script
          dangerouslySetInnerHTML={{
            __html:
              'try{var t=localStorage.getItem("wx-theme");if(t==="light"||t==="dark"){document.documentElement.dataset.theme=t}}catch(e){}',
          }}
        />
        <script
          dangerouslySetInnerHTML={{
            __html: JSON.stringify(jsonLd).replace(/</g, "\\u003c"),
          }}
          type="application/ld+json"
        />
      </head>
      <body>{children}</body>
    </html>
  );
}
