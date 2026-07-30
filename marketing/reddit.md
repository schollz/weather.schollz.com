# Reddit launch material

Reddit should be a conversation and feedback channel, not a synchronized link
drop. Post to one community at a time, answer every good-faith comment, and use
what you learn before trying the next community.

## Channel priority

| Priority | Community | Fit | Use when |
| --- | --- | --- | --- |
| 1 | [r/SideProject](https://www.reddit.com/r/SideProject/) | High | The account is in normal standing and you can stay to discuss the build. The community explicitly exists for sharing side projects and feedback. |
| 2 after August 25 | [r/commandline](https://www.reddit.com/r/commandline/) | High audience fit, currently ineligible | The project is older than 30 days, you can compare it honestly with alternatives, and **you write the title and post yourself**. Its current rules prohibit AI-generated post text and titles. |
| Defer until at least mid-October | [r/InternetIsBeautiful](https://www.reddit.com/r/InternetIsBeautiful/) | Good product fit, poor timing now | At least 90% of the account's recent activity is unrelated to your own sites, no similar weather site has appeared recently, and you are comfortable with strict moderator discretion. |
| 4 | [r/webdev](https://www.reddit.com/r/webdev/) | Conditional | On a Saturday only, using the Showoff Saturday flair, with a technical build story rather than marketing copy. Current rules reject LLM-generated or low-effort material, so write it yourself. |
| Do not use yet | [r/opensource](https://www.reddit.com/r/opensource/) | Not currently eligible | The repository has no `LICENSE` file. The community requires linked repositories to contain an OSI-approved license. |

Always re-read the visible rules on the day of submission. Do not cross-post the
same copy across communities.

The repository's first commit is July 25, 2026, so r/commandline's “older than
30 days” rule makes August 26 a conservative first posting date. As of July 30,
r/InternetIsBeautiful had featured wttr.in on July 11 and another minimalist
weather site on April 1. Waiting until at least mid-October is a strategic
recommendation, not a written subreddit rule; recheck the feed before using the
copy below.

## r/SideProject

Suggested post type: text post with the browser-focused gallery image or a live
site screenshot attached.

**Title**

```text
wthrtxt.com – local weather without the clutter, in a browser or curl
```

**Body**

```text
I wanted a weather page that answers “what is it doing outside?” and then gets out of the way. Most weather sites kept adding news, video, giant maps, and more navigation, so I built wthrtxt.com.

It shows:

- current conditions and hourly detail
- rainfall, humidity, and wind
- a seven-day forecast
- daily record highs/lows and historical averages
- a readable URL for every place

The slightly unusual part is that the same URL is also the terminal interface:

curl https://wthrtxt.com/seattle

U.S. locations use NOAA forecasts and observations plus ACIS climate history. Other locations use Open-Meteo. It is free, needs no account, and the source is here:
https://github.com/schollz/wthrtxt

Try it:
https://wthrtxt.com

One thing I’m unsure about: are the daily records genuinely useful in the seven-day table, or do they make it too dense? I’d appreciate blunt feedback on that balance.
```

The specific question is intentional. It gives readers something more useful
to discuss than “thoughts?”

If AI materially contributed to the code or design, add a short, accurate
disclosure in your own words. Do not imply the project was entirely hand-coded
if that is not true.

## r/InternetIsBeautiful

This community is a plausible eventual fit because wthrtxt is a single-purpose,
minimal, free website with no account requirement. It is a poor fit **right
now** because wttr.in appeared there on July 11, 2026 and another minimalist
weather site appeared on April 1. Moderators reject sites they consider
insufficiently unique or too similar to a recent submission. The browser/curl
duality and climate-record rows are the strongest differentiators, but waiting
is safer.

Before posting:

- Confirm at least 90% of the account's recent Reddit participation is unrelated
  to sites you own or operate.
- Search the community for `wthrtxt`, `weather`, and `curl` to avoid a recent
  duplicate.
- Wait until at least mid-October 2026, then reconsider based on the current
  feed. A meaningful product improvement is a better reason to post than time
  alone.
- Confirm the site is fast under load.
- Submit the direct top-level URL, not GitHub, an image, an article, or a
  Product Hunt page.

**Link-post title**

```text
A minimal local weather site that also turns into plain text when opened with curl
```

**URL**

```text
https://wthrtxt.com
```

**First comment**

```text
Disclosure: I made this. I wanted the useful density of a terminal forecast without giving up a readable browser view. It is free, needs no account, and uses NOAA/ACIS for supported U.S. locations and Open-Meteo elsewhere. I would be interested to hear whether the records and averages are useful or simply too much information.
```

If the account does not satisfy the community's 90/10 rule, do not submit. Spend
time participating normally; do not manufacture low-value comments to hit a
ratio.

## r/commandline: user-written brief

Do not paste AI-written marketing copy into r/commandline. Its current rule 4
explicitly prohibits AI-generated post text and titles. Write the post in your
own words from this factual brief:

- Do not post before August 26, 2026; the first repository commit was July 25
  and the community rejects projects newer than 30 days.
- State immediately that you made wthrtxt.
- Show `curl https://wthrtxt.com/seattle`.
- Explain content negotiation: terminal clients receive plain text; browsers
  receive the visual reader; `?format=text` and `?format=html` override it.
- Mention readable place slugs and coordinate URLs.
- Explain the data sources: NOAA and ACIS in supported U.S. areas, Open-Meteo
  elsewhere.
- Include project age if it is over 30 days; the community rejects newer
  projects.
- Link both `https://wthrtxt.com` and
  `https://github.com/schollz/wthrtxt`.
- Acknowledge alternatives including `wttr.in`.
- Explain the difference without dismissing alternatives: browser and terminal
  share one location URL; the seven-day table includes daily records and
  averages; U.S. output uses NOAA forecasts and observations.
- Disclose any material use of AI in the project exactly as the community's
  current rules require.
- Ask a terminal-specific question, such as which columns users would remove or
  whether JSON output would be useful.

Use the `Command Line Interface` flair. Re-read all rules before submitting,
especially the AI policy, project-age requirement, and alternative-software
requirement.

## r/webdev: optional user-written brief

Only consider this after the main launch, on a Saturday, with the `Showoff
Saturday` flair. Write it yourself and make it a technical post, not a launch
announcement.

Possible technical spine:

- One URL serving a React browser experience and Go-rendered plaintext through
  content negotiation.
- The cache boundary between NOAA/Open-Meteo data and the frontend.
- Worldwide place search plus readable location slugs.
- What was difficult about normalizing precipitation and daily history across
  providers.
- A specific request for feedback on accessibility, responsive tables, or
  content negotiation.

Link the source and live site. Do not reuse the r/SideProject body.

## Reddit reply bank

Adapt these to the actual conversation rather than pasting the same reply
repeatedly.

**“How is this different from wttr.in?”**

```text
wttr.in is the obvious comparison and helped make terminal weather familiar. My focus here is one location URL that is equally usable as a browser reader or plain text, plus NOAA/ACIS data for supported U.S. places and daily records/averages in the seven-day view. If you use wttr.in regularly, I’d be interested in what wthrtxt would need to earn a place beside it.
```

**“Why do I need another weather site?”**

```text
You may not, honestly. I built it for people who want a compact forecast without the surrounding weather-portal content, and for people who want the exact same location URL in a browser and terminal. The records and historical averages are the other part I could not find in the format I wanted.
```

**“Where does the data come from?”**

```text
Supported U.S. locations use NOAA forecasts and station observations, with ACIS for record highs/lows and averages. Other locations use Open-Meteo forecasts and ERA5-Land estimates for history. The site labels estimated history separately from official station observations.
```

**“Is it open source?”**

```text
The source is public on GitHub. I have not added an explicit license yet, so I’m not calling it open source until I make that licensing decision.
```

**Bug report**

```text
Thanks—that is useful. Could you share the location URL, approximate time, browser or curl command, and the value that looked wrong? I’ll compare it against the provider response.
```

**Feature request**

```text
I can see how that would help for [use case]. The constraint I’m trying to keep is a focused default view. Would [smallest version] solve most of it for you?
```

## Posting cadence

- Do not publish the r/SideProject and r/InternetIsBeautiful posts on the same
  day.
- Wait at least 48 hours and respond to the first discussion before making a
  second submission.
- Do not put a Product Hunt link into the Reddit product posts; send readers
  directly to the useful site.
- If a post is removed, read the moderator reason. Do not repost unchanged or
  argue publicly.
- Follow up only when there is a real improvement or a useful account of what
  you learned.
