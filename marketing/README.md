# wthrtxt.com launch kit

Prepared July 30, 2026.

This folder contains a practical launch campaign for
[wthrtxt.com](https://wthrtxt.com). The main goal is to find repeat users and
useful product feedback, not to chase a one-day leaderboard position.

## Positioning

**Core promise:** Local weather without the clutter—in a browser or curl.

**Best first audiences:**

1. People who want a quick forecast without a weather portal, news feed, or
   account.
2. Developers and terminal users who want a readable forecast from `curl`.
3. Weather-data enthusiasts who value NOAA forecasts, rainfall detail, and
   daily temperature records and averages.

**Reasons to believe:**

- Current conditions, hourly detail, and a seven-day forecast.
- Rainfall, humidity, wind, daily temperature records, and historical
  averages.
- NOAA and ACIS data for supported U.S. locations; Open-Meteo worldwide.
- Readable, shareable place URLs.
- The same URL returns a plain-text forecast to terminal clients.
- Free and usable without an account.

**Differentiation:** Mainstream weather sites optimize for many destinations
and content types. wthrtxt optimizes for one task: reading the local forecast
quickly. Unlike a terminal-only weather service, it is also a polished browser
reader; unlike a browser-only forecast, every location works with `curl`.

Do not claim that the project is open source yet. The source is public on
GitHub, but the repository currently has no `LICENSE` file. Add an OSI-approved
license first if that is the intention.

## Recommended sequence

| When | Channel | Action | Goal |
| --- | --- | --- | --- |
| Days -14 to -8 | Product Hunt | Complete the personal profile, join discussions, and leave useful feedback on products you genuinely try. | Become a real participant before launching. |
| Days -10 to -7 | Reddit | Participate normally in the target communities. Check the account's recent activity against each community's self-promotion rules. | Avoid a drive-by promotional profile. |
| Day -7 | r/SideProject | Publish the feedback-led post in [reddit.md](reddit.md). | Learn which part of the pitch resonates. |
| Days -6 to -3 | Product | Fix any launch-blocking bugs or confusing copy discovered on Reddit. Do not add marginal features. | Improve first-use success. |
| Day -3 | Product Hunt | Upload the assets and schedule the launch. | Remove launch-day setup work. |
| Day -1 | Product Hunt | Recheck the live URL, mobile view, location permission fallback, search, `curl`, social image, and analytics. | Prevent a broken first impression. |
| Day 0, 12:01 a.m. PT | Product Hunt | Launch and add the maker comment. | Get a full Product Hunt day. |
| Day 0 | Product Hunt | Reply promptly and specifically to every substantive comment. Ask for feedback, never for upvotes. | Turn visits into conversation. |
| Day +7 | Review | Compare qualified visits, repeat visits, place searches, `curl` traffic, and feedback themes by source. | Decide where to keep investing. |
| Day +14 | Follow-up | Share a short “what changed after launch” update in r/SideProject if there are meaningful changes. | Close the feedback loop. |
| August 26 or later | r/commandline | Use the factual brief in [reddit.md](reddit.md), but write the title and body yourself. | Reach terminal users after satisfying the community's 30-day project-age rule. |
| Mid-October or later | r/InternetIsBeautiful | Recheck for recent weather submissions, the account's 90/10 history, and a meaningful differentiator before posting. | Reach minimal-web fans without colliding with recent similar posts. |

Product Hunt says weekend launches receive 15% more “Visit” clicks than weekday
launches and may suit personal apps and side projects. A Saturday launch is a
good default here if you can be present to reply; otherwise choose the day on
which you can be responsive for most of the Pacific-time cycle. Product Hunt's
standard scheduling guidance is 12:01 a.m. Pacific.

## Success measures

Use a seven-day window after each post. Record:

- Unique visitors by referrer.
- Visitors who search for a place or approve location access.
- Return visitors within seven days.
- Terminal/plain-text requests.
- GitHub visits, stars, issues, and useful feedback.
- Product Hunt comments and reviews, not only rank or upvotes.

A sensible first target is **100 qualified visits, 15 return visitors, and 10
pieces of specific feedback** across the campaign. These are learning targets,
not claims about expected conversion.

## Files

- [product-hunt.md](product-hunt.md): listing fields, maker comment, sharing
  copy, reply bank, and upload checklist.
- [reddit.md](reddit.md): subreddit order, ready-to-paste posts, rule
  constraints, and reply bank.
- [assets/product-hunt-thumbnail.png](assets/product-hunt-thumbnail.png):
  240×240 Product Hunt thumbnail.
- [assets/product-hunt-gallery-01.png](assets/product-hunt-gallery-01.png):
  1270×760 browser-focused gallery image.
- [assets/product-hunt-gallery-02.png](assets/product-hunt-gallery-02.png):
  1270×760 terminal-focused gallery image.

## Sources checked

- [Product Hunt launch guide](https://www.producthunt.com/launch)
- [Product Hunt preparation and content checklist](https://www.producthunt.com/launch/preparing-for-launch)
- [Product Hunt launch-sharing rules](https://www.producthunt.com/launch/sharing-your-launch)
- [Product Hunt featuring guidelines](https://help.producthunt.com/en/articles/9883485-product-hunt-featuring-guidelines)
- [r/SideProject](https://www.reddit.com/r/SideProject/)
- [r/InternetIsBeautiful rules](https://www.reddit.com/r/InternetIsBeautiful/)
- [r/commandline rules](https://www.reddit.com/r/commandline/)
- [r/webdev rules](https://www.reddit.com/r/webdev/)
- [r/opensource rules](https://www.reddit.com/r/opensource/)

Community rules change. Re-read the visible rules immediately before posting.
