import type { Metadata, Viewport } from "next";
import "./globals.css";
import {
  ROOT_SOCIAL_TITLE,
  ROOT_TITLE,
  SITE_DESCRIPTION,
  SITE_NAME,
  SITE_URL,
  SOCIAL_IMAGE,
  siteJsonLd,
} from "./site";

export const dynamic = "force-static";

export const metadata: Metadata = {
  alternates: {
    canonical: "/",
  },
  applicationName: SITE_NAME,
  appleWebApp: {
    capable: true,
    statusBarStyle: "black-translucent",
    title: SITE_NAME,
  },
  authors: [{ name: SITE_NAME, url: SITE_URL }],
  category: "weather forecast",
  creator: SITE_NAME,
  description: SITE_DESCRIPTION,
  formatDetection: {
    address: false,
    email: false,
    telephone: false,
  },
  icons: {
    icon: [{ type: "image/svg+xml", url: "/favicon.svg" }],
    shortcut: ["/favicon.svg"],
  },
  keywords: [
    "local weather forecast",
    "current weather conditions",
    "hourly forecast",
    "7-day weather forecast",
    "worldwide weather forecast",
    "temperature humidity rain",
    "weather by curl",
    "plaintext weather",
    "NOAA weather",
    "Open-Meteo weather",
    "daily temperature records",
    "ACIS",
  ],
  manifest: "/manifest.webmanifest",
  metadataBase: new URL(SITE_URL),
  openGraph: {
    description: SITE_DESCRIPTION,
    images: [SOCIAL_IMAGE],
    locale: "en_US",
    siteName: SITE_NAME,
    title: ROOT_SOCIAL_TITLE,
    type: "website",
    url: "/",
  },
  publisher: SITE_NAME,
  referrer: "origin-when-cross-origin",
  robots: {
    follow: true,
    googleBot: {
      follow: true,
      index: true,
      "max-image-preview": "large",
      "max-snippet": -1,
      "max-video-preview": -1,
    },
    index: true,
  },
  title: {
    default: ROOT_TITLE,
    template: `%s | ${SITE_NAME}`,
  },
  twitter: {
    card: "summary_large_image",
    description: SITE_DESCRIPTION,
    images: [{ alt: SOCIAL_IMAGE.alt, url: SOCIAL_IMAGE.url }],
    title: ROOT_SOCIAL_TITLE,
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

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html:
              'try{var t=localStorage.getItem("wx-theme");if(t==="light"||t==="dark"){document.documentElement.dataset.theme=t}}catch(e){}',
          }}
        />
        <script
          dangerouslySetInnerHTML={{
            __html: JSON.stringify(siteJsonLd).replace(/</g, "\\u003c"),
          }}
          id="site-json-ld"
          type="application/ld+json"
        />
      </head>
      <body>{children}</body>
    </html>
  );
}
