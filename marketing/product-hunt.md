# Product Hunt launch material

All copy below is ready to paste unless a note says otherwise.

## Listing fields

**Product name**

```text
wthrtxt.com
```

**Product URL**

```text
https://wthrtxt.com
```

Use the direct URL. Product Hunt does not accept tracking parameters in this
field.

**Tagline — 52 characters**

```text
Local weather without the clutter—in browser or curl
```

**Description — 422 characters**

```text
wthrtxt.com is a fast, free weather reader for people who want the forecast, not a weather portal. Search any place to see current conditions, hourly details, rainfall, a seven-day outlook, and daily temperature records and historical averages. U.S. forecasts use NOAA and ACIS; worldwide forecasts use Open-Meteo. Every location has a readable URL, and the same forecast works as plain text with curl—no account required.
```

**Pricing**

```text
Free
```

**Launch tags**

Choose the closest current matches to:

1. Weather
2. Developer Tools
3. Web App

If “Weather” is unavailable, use “Productivity.” If an “Open Source” tag
appears, do not select it until the repository has an explicit open-source
license.

**Maker**

Mark yourself as the maker. Product Hunt says there is no discernible advantage
to paying or recruiting a third-party hunter.

**Shoutouts**

If these products are available in the Product Hunt selector, use the three
that mattered most to the build:

- Open-Meteo
- Next.js
- NOAA / National Weather Service

Do not force unrelated shoutouts merely to fill the fields.

**Thumbnail**

Upload [assets/product-hunt-thumbnail.png](assets/product-hunt-thumbnail.png).

**Gallery order**

1. [assets/product-hunt-gallery-01.png](assets/product-hunt-gallery-01.png)
2. [assets/product-hunt-gallery-02.png](assets/product-hunt-gallery-02.png)

The product is simple enough that a video is optional. A weak explainer video
would add less than the two clear images and the live product.

## First maker comment

```text
Hi Product Hunt — I built wthrtxt because checking the weather had started to feel like visiting a media portal. I wanted one small page that answers the practical questions quickly, then gets out of the way.

wthrtxt shows:

• current conditions and the next several hours
• rainfall, humidity, and wind
• a seven-day forecast
• daily record highs, record lows, and historical averages
• a readable URL for every place

There is also a second interface hiding in plain sight:

curl https://wthrtxt.com/seattle

The server returns the same forecast as plain text for curl, wget, HTTPie, and PowerShell. U.S. locations use NOAA forecasts and observations plus ACIS climate history; other locations use Open-Meteo.

It is free and needs no account. I would especially value feedback on two things: does the page show the right amount of information, and is the terminal format something you would actually use?
```

## Personal profile copy

Product Hunt requires a personal rather than a company account. Adapt the
bracketed text before using this.

**Headline**

```text
Independent maker of small, focused web tools
```

**About**

```text
I make small tools that remove steps from everyday tasks. I’m currently working on wthrtxt.com, a clutter-free local weather reader for browsers and terminals, alongside croc, cowyo, and yesnotice.
```

**Website**

```text
https://wthrtxt.com/about/
```

## Launch announcement

Use this on a personal social account or with people who already know your
work. Replace `[PRODUCT HUNT URL]` after the launch is live.

```text
I launched wthrtxt.com on Product Hunt today.

It is local weather without the weather-site clutter: current conditions, hourly detail, a seven-day outlook, and temperature records in a browser—or the same forecast from curl.

If you try it, I’d love to hear what feels missing or unnecessary:
[PRODUCT HUNT URL]
```

Do not change the last line to an upvote request. Product Hunt explicitly
prohibits asking people to upvote.

## Direct note to existing users or friends

Send this only to people with whom you already have a real relationship. Do not
bulk-send it to strangers.

```text
I’m launching wthrtxt.com on Product Hunt today. It’s the minimal weather reader I’ve been building—the same URL works in a browser and through curl.

Would you be willing to try your location and tell me whether anything is confusing or wrong? Comments on the Product Hunt page are useful, but no pressure:
[PRODUCT HUNT URL]
```

## Reply bank

Customize every reply to the actual comment.

**When someone likes the minimal design**

```text
Thank you. Deciding what not to show has been most of the design work. Is there any remaining section you would collapse or remove?
```

**When someone mentions wttr.in or another alternative**

```text
Yes—wttr.in helped establish that weather belongs in the terminal. My focus with wthrtxt is a shared browser-and-terminal experience, readable location URLs, and daily records and historical averages alongside the forecast. I’m interested in where the terminal output still falls short for your workflow.
```

**When someone requests a feature**

```text
That makes sense for [their use case]. I’m trying to protect the focused default view, but I can see [feature] working as [an optional or compact form]. What is the smallest version that would solve it for you?
```

**When someone reports incorrect weather**

```text
Thanks for the precise report. Could you share the location URL, approximate time, and which value looked wrong? I’ll compare it with the underlying provider and follow up here.
```

**When someone asks about data sources**

```text
Supported U.S. locations use NOAA forecasts and station observations, plus ACIS for daily climate records and averages. Other locations use Open-Meteo forecasts and ERA5-Land estimates for climate history. The About page labels the distinction because model-based history is not the same as an official station observation.
```

**When someone asks about privacy**

```text
There is no account requirement. In the browser, location is requested through the browser permission prompt, and place search is also available if you prefer not to share precise location. The About page lists every external data source the browser uses.
```

Do not claim “no tracking” unless the production analytics configuration has
been checked and that statement is accurate.

## Scheduling and launch-day checklist

Product Hunt currently allows scheduling up to one month ahead. Its standard
advice is 12:01 a.m. Pacific so the product receives a full homepage cycle.
Weekends may be a good fit for a solo side project, but responsiveness matters
more than a theoretical perfect day.

- [ ] Product Hunt account is at least one week old and the profile is complete.
- [ ] You have used Product Hunt as a genuine participant before launch.
- [ ] The homepage, search, geolocation fallback, readable location URL, and
      `curl` output work in production.
- [ ] The social preview renders correctly.
- [ ] Thumbnail and both gallery images are uploaded in the listed order.
- [ ] The first maker comment is loaded into the scheduled launch.
- [ ] The launch is scheduled for a day you can actively monitor.
- [ ] Analytics can distinguish Product Hunt and Reddit referrers.
- [ ] You have not asked anyone for an upvote.
- [ ] You are ready to answer comments and bug reports throughout the day.

## Product Hunt-specific objective

Optimize for:

1. Specific feedback about information density.
2. Evidence that people return or use the terminal interface.
3. Reviews and quotes that explain the value in a user's own words.

Treat Product of the Day as a possible outcome, not the campaign goal.
