// English is the source locale: its shape defines the Messages type, so every
// other locale must provide exactly these keys.
const plural = new Intl.PluralRules("en");
const count = (n: number, one: string, other: string) =>
  `${n} ${plural.select(n) === "one" ? one : other}`;

export const en = {
  site: {
    // Placeholder working name; change it here only.
    name: "Ground News Taiwan",
    tagline: "One event, every outlet's coverage",
    description:
      "Every outlet's coverage of the same event, side by side, with a short summary of each article and a timeline of the event. The site never labels or scores an outlet or an article; readers compare for themselves.",
  },
  nav: {
    label: "Main",
    outlets: "Outlets",
    about: "About",
    skipToContent: "Skip to content",
  },
  settings: {
    label: "Settings",
    appearance: "Appearance",
    system: "System",
    light: "Light",
    dark: "Dark",
    language: "Language",
  },
  common: {
    opensInNewTab: "(opens in a new tab)",
    loading: "Loading page",
  },
  home: {
    title: "Current events",
    empty: "No events yet.",
    loadMore: "Load more",
    loadingMore: "Loading…",
  },
  eventCard: {
    outlets: (n: number) => count(n, "outlet", "outlets"),
    articles: (n: number) => count(n, "article", "articles"),
    updated: "Updated",
    since: "Since",
  },
  event: {
    firstSeen: "First reported",
    updated: "Updated",
    timeline: "Timeline",
    coverage: "Coverage",
    coverageCount: (articles: number, outlets: number) =>
      `${count(articles, "article", "articles")} from ${count(outlets, "outlet", "outlets")}`,
    newestFirst: "Newest first",
    oldestFirst: "Oldest first",
    sortLabel: "Sort coverage",
    description: (title: string, latest: string) =>
      latest ? `${title}. Latest: ${latest}` : `How each outlet covered: ${title}`,
  },
  timeline: {
    empty: "No developments have been identified for this event yet.",
    approximate: "c.",
    reportedBy: "Outlets that reported this development",
    showEarlier: (n: number) => `Show ${count(n, "earlier development", "earlier developments")}`,
    hideEarlier: "Hide earlier developments",
    gapMonths: (n: number) => `${count(n, "month", "months")} later`,
    gapYears: (years: number, months: number) =>
      months > 0
        ? `${count(years, "year", "years")} and ${count(months, "month", "months")} later`
        : `${count(years, "year", "years")} later`,
  },
  article: {
    unknownDate: "Publication time unknown",
    readOriginal: "Read the original",
    headlineOnly:
      "The full article has not been retrieved yet. The headline is shown as published.",
    alsoRunBy: (n: number) => `Also run by ${count(n, "other outlet", "other outlets")}`,
  },
  outlets: {
    title: "Outlets",
    intro:
      "The news sources this site follows, in alphabetical order. The site does not describe, group or rate them.",
  },
  about: {
    title: "About",
    sections: [
      {
        heading: "What this site does",
        body: [
          "When something happens, many outlets report it, each in its own way. This site gathers every outlet's coverage of the same event on one page, with a short summary of each article and a timeline of what happened.",
        ],
      },
      {
        heading: "No labels, no scores",
        body: [
          "The site never labels, scores or judges an outlet or an article. Every article is shown the same way, in order of time. Comparing the coverage, and deciding what to make of it, is left to you.",
        ],
      },
      {
        heading: "How summaries are written",
        body: [
          "Each summary is written by an AI model from that one article only. It keeps what the article emphasises, names who it quotes and keeps the article's own terms, without correcting, balancing or commenting on it. Summaries are short and in our own words; they point to the original and do not replace it. Headlines are shown exactly as the outlet published them.",
        ],
      },
      {
        heading: "How the timeline is built",
        body: [
          'The timeline tracks what happened in the event, not when articles were published. It is the one place the site speaks in its own voice, so each step is a single short, neutral line, checked against its source articles. Accusations stay attributed until a court has ruled. A date marked "c." is approximate.',
        ],
      },
      {
        heading: "Outlets that don't allow AI summaries",
        body: [
          "Some outlets say their articles may not be read by AI. For those, the site never opens the article and never gives it to a model: it lists only the headline, exactly as published, with a link, the way a search engine would. They are matched to events by the names in their headlines, which is less reliable, so you will see fewer of their articles here than they publish.",
        ],
      },
      {
        heading: "Mistakes",
        body: [
          "An automated system will sometimes group articles wrongly or summarise one badly. If you see an error, please tell us so it can be fixed.",
        ],
      },
    ],
  },
  admin: {
    title: "Admin",
    loginTitle: "Admin sign-in",
    token: "Admin token",
    tokenHelp: "The ADMIN_TOKEN value configured on the API server.",
    signIn: "Sign in",
    signOut: "Sign out",
    badToken: "That token was not accepted, or the admin tool is disabled on the server.",
    linksTitle: "Link decisions",
    linksIntro:
      "Articles the model linked to an event, least confident first. Open an event to merge it or to move an article.",
    noLinks: "No link decisions yet.",
    colArticle: "Article",
    colOutlet: "Outlet",
    colEvent: "Event",
    colConfidence: "Confidence",
    colPublished: "Published",
    colAction: "Action",
    openEvent: "Review event",
    publicPage: "Public page",
    backToLinks: "Back to link decisions",
    mergeButton: "Merge another event into this one",
    mergeTitle: "Merge an event into this one",
    mergeDescription:
      "Every article of the event you pick moves here, and that event is deleted. This cannot be undone.",
    merge: "Merge",
    moveButton: "Move",
    moveTitle: "Move article to another event",
    moveDescription:
      "The article and its reprints move. If it was a timeline step that other articles attach to, the earliest of them takes the step over.",
    move: "Move",
    cancel: "Cancel",
    searchEvents: "Search events by title or id",
    searchHelp: "Type at least two characters.",
    noMatches: "No matching events.",
    orNewEvent: "Or start a new event",
    newEventTitle: "Title for the new event",
    newEventHelp: "Short, factual and neutral; it should still fit as the story develops.",
    articles: (n: number) => count(n, "article", "articles"),
    merged: "The events were merged.",
    moved: "The article was moved.",
    reprintNote: "reprint",
  },
  error: {
    notFoundTitle: "Page not found",
    notFoundBody:
      "The page you asked for doesn't exist, or the event has been merged into another.",
    unavailableTitle: "Temporarily unavailable",
    unavailableBody: "The site can't reach its data right now. Please try again in a moment.",
    genericTitle: "Something went wrong",
    genericBody: "An unexpected error occurred.",
    backHome: "Back to current events",
  },
};

export type Messages = typeof en;
