import type { Messages } from "./en";

export const zhTW: Messages = {
  site: {
    name: "新聞並陳",
    tagline: "同一事件，各家媒體怎麼報導",
    description:
      "把各家媒體對同一事件的報導並列，附上每篇報導的摘要與事件時間軸。本站不為任何媒體或報導貼標籤或評分，由讀者自行比較。",
  },
  nav: {
    label: "主要導覽",
    outlets: "媒體",
    about: "關於",
    skipToContent: "跳到主要內容",
  },
  settings: {
    label: "設定",
    appearance: "外觀",
    system: "跟隨系統",
    light: "淺色",
    dark: "深色",
    language: "語言",
  },
  common: {
    opensInNewTab: "（在新分頁開啟）",
    loading: "頁面載入中",
  },
  home: {
    title: "目前的事件",
    empty: "目前還沒有事件。",
    loadMore: "載入更多",
    loadingMore: "載入中…",
  },
  eventCard: {
    outlets: (n) => `${n} 家媒體`,
    articles: (n) => `${n} 篇報導`,
    updated: "更新於",
    since: "始於",
  },
  event: {
    firstSeen: "首次報導",
    updated: "更新於",
    timeline: "時間軸",
    coverage: "各家報導",
    coverageCount: (articles, outlets) => `${outlets} 家媒體、${articles} 篇報導`,
    newestFirst: "由新到舊",
    oldestFirst: "由舊到新",
    sortLabel: "報導排序",
    description: (title, latest) =>
      latest ? `${title}。最新進展：${latest}` : `各家媒體如何報導：${title}`,
  },
  timeline: {
    empty: "這個事件目前還沒有整理出進展。",
    approximate: "約",
    reportedBy: "報導此進展的媒體",
    showEarlier: (n) => `顯示較早的 ${n} 則進展`,
    hideEarlier: "收合較早的進展",
    gapMonths: (n) => `${n} 個月後`,
    gapYears: (years, months) => (months > 0 ? `${years} 年 ${months} 個月後` : `${years} 年後`),
  },
  article: {
    unknownDate: "發布時間未知",
    readOriginal: "閱讀原文",
    headlineOnly: "尚未取得完整內容。標題依原樣呈現。",
    alsoRunBy: (n) => `另有 ${n} 家媒體刊登`,
  },
  outlets: {
    title: "媒體",
    intro: "本站追蹤的新聞來源，依名稱排序。本站不描述、不分類，也不評價這些媒體。",
  },
  about: {
    title: "關於",
    sections: [
      {
        heading: "這個網站做什麼",
        body: [
          "一件事發生後，許多媒體都會報導，各有各的寫法。本站把各家媒體對同一事件的報導集中在同一頁，附上每篇報導的簡短摘要，以及事件經過的時間軸。",
        ],
      },
      {
        heading: "不貼標籤、不評分",
        body: [
          "本站不為任何媒體或報導貼標籤、評分或下判斷。每篇報導都以相同方式呈現，依時間排序。如何比較、如何解讀，由你決定。",
        ],
      },
      {
        heading: "摘要怎麼寫",
        body: [
          "每則摘要由 AI 模型只根據該篇報導撰寫，保留報導強調的重點、引述的對象與原本的用詞，不更正、不平衡、不評論。摘要簡短並以我們自己的文字寫成，用來指向原文，而不是取代原文。標題則完全依照媒體刊出的樣子呈現。",
        ],
      },
      {
        heading: "時間軸怎麼整理",
        body: [
          "時間軸記錄的是事件本身的進展，而不是報導的發布時間。這是本站唯一以自己口吻說話的地方，因此每一步都只有一行簡短、中性的敘述，並與來源報導核對。在法院判決之前，指控一律註明出處。標示「約」的日期為推估。",
        ],
      },
      {
        heading: "不允許 AI 摘要的媒體",
        body: [
          "有些媒體表明其報導不得由 AI 讀取。對這些媒體，本站不會開啟報導內文，也不會把內容交給模型：只像搜尋引擎一樣，列出原標題與連結。這類報導是依標題中的人名、機構與地名歸入事件，準確度較低，因此你在這裡看到的篇數會少於它們實際刊出的數量。",
        ],
      },
      {
        heading: "錯誤",
        body: [
          "自動化系統有時會把報導分錯事件，或把某篇摘要寫壞。如果你發現錯誤，請告訴我們，我們會修正。",
        ],
      },
    ],
  },
  admin: {
    title: "管理",
    loginTitle: "管理員登入",
    token: "管理權杖",
    tokenHelp: "API 伺服器上設定的 ADMIN_TOKEN。",
    signIn: "登入",
    signOut: "登出",
    badToken: "權杖不正確，或伺服器未啟用管理工具。",
    linksTitle: "歸併判斷",
    linksIntro: "模型把報導歸入事件的判斷，信心最低的排在最前面。開啟事件即可合併事件或移動報導。",
    noLinks: "目前還沒有歸併判斷。",
    colArticle: "報導",
    colOutlet: "媒體",
    colEvent: "事件",
    colConfidence: "信心",
    colPublished: "發布時間",
    colAction: "操作",
    openEvent: "檢視事件",
    publicPage: "公開頁面",
    backToLinks: "回到歸併判斷",
    mergeButton: "把另一個事件併入這個事件",
    mergeTitle: "把事件併入這個事件",
    mergeDescription: "你選的事件的所有報導都會移到這裡，該事件會被刪除，且無法復原。",
    merge: "合併",
    moveButton: "移動",
    moveTitle: "把報導移到另一個事件",
    moveDescription:
      "報導與其轉載會一起移動。如果它是其他報導所依附的時間軸節點，最早的那篇會接手該節點。",
    move: "移動",
    cancel: "取消",
    searchEvents: "以標題或編號搜尋事件",
    searchHelp: "請至少輸入兩個字元。",
    noMatches: "找不到符合的事件。",
    orNewEvent: "或建立新事件",
    newEventTitle: "新事件的標題",
    newEventHelp: "簡短、符合事實且中性；事件後續發展時仍應適用。",
    articles: (n) => `${n} 篇報導`,
    merged: "事件已合併。",
    moved: "報導已移動。",
    reprintNote: "轉載",
  },
  error: {
    notFoundTitle: "找不到頁面",
    notFoundBody: "你要找的頁面不存在，或該事件已併入其他事件。",
    unavailableTitle: "暫時無法使用",
    unavailableBody: "網站目前無法取得資料，請稍後再試。",
    genericTitle: "發生錯誤",
    genericBody: "發生未預期的錯誤。",
    backHome: "回到目前的事件",
  },
};
