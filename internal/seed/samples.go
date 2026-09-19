package seed

import "time"

// The sample events are fictional and clearly marked as such. They exist so
// the site can be developed without crawling or an OpenAI key, and they are
// never attributed to a real article: every URL points at example.com.
//
// Headlines stay in Chinese, as real ones would (the site shows original
// headlines unchanged). Titles, timeline lines and summaries are the site's
// own text and are in English.

const sampleURLPrefix = "https://example.com/ground-sample/"

type sampleEvent struct {
	Title    string
	Articles []sampleArticle
}

type sampleArticle struct {
	Key    string // unique within the seed; becomes the URL suffix
	Outlet string // outlet slug
	// Ago is how long before "now" the article was published.
	Ago      time.Duration
	Headline string
	Summary  string

	// Development is set on the first article to report a step.
	Development string
	// HappenedAgo is how long ago the development happened; nil means unclear.
	HappenedAgo *time.Duration
	Approximate bool
	// StepOf is the Key of the article that first reported the same development.
	StepOf string
	// ReprintOf is the Key of the wire original this article reprints.
	ReprintOf string
}

const day = 24 * time.Hour

func ago(d time.Duration) *time.Duration { return &d }

var samples = []sampleEvent{
	{
		// A long-running case that went dormant and came back: steps months apart.
		Title: "[Sample] Drug case against a man surnamed Wang in Sample City",
		Articles: []sampleArticle{
			{
				Key: "drug-1-cna", Outlet: "cna", Ago: 190 * day,
				Headline:    "【範例】範例市警方臨檢查獲王姓男子疑持有毒品",
				Summary:     "The report says Sample City police found items suspected to be drugs in Wang's car during a routine roadside check. Police said the items had been sent for testing and that Wang was calm when taken in. The article quotes police and does not include a response from Wang.",
				Development: "Police find suspected drugs in Wang's car during a roadside check",
				HappenedAgo: ago(190 * day),
			},
			{
				Key: "drug-1-ettoday", Outlet: "ettoday", Ago: 190*day - 3*time.Hour,
				Headline: "【範例】臨檢攔下轎車 警見駕駛神色緊張一查發現毒品",
				Summary:  "The article centres on the roadside check itself, describing how officers asked to open the boot after noticing the driver \"looked nervous\". It quotes an officer saying several packets of suspected drugs were found, and notes that Wang has no related prior record.",
				StepOf:   "drug-1-cna",
			},
			{
				Key: "drug-2-ltn", Outlet: "ltn", Ago: 188 * day,
				Headline:    "【範例】王男涉毒遭逮 警詢後移送地檢署",
				Summary:     "The article says police arrested Wang after the test results came back and referred him to prosecutors for alleged violation of the Narcotics Hazard Prevention Act. It quotes police alleging that Wang admitted to part of the conduct, while his lawyer said the investigation would clarify the facts.",
				Development: "Police arrest Wang and refer him to prosecutors",
				HappenedAgo: ago(188 * day),
			},
			{
				Key: "drug-2-udn", Outlet: "udn", Ago: 188*day - 5*time.Hour,
				Headline: "【範例】涉持有毒品 王男移送檢方複訊後交保",
				Summary:  "The report says Wang was referred to the district prosecutors' office and released on bail after questioning. It recaps the earlier roadside check in which police found suspected drugs, and quotes prosecutors saying the case remains under investigation.",
				StepOf:   "drug-2-ltn",
			},
			{
				Key: "drug-3-cna", Outlet: "cna", Ago: 95 * day,
				Headline:    "【範例】檢方依毒品危害防制條例起訴王姓男子",
				Summary:     "The report says prosecutors concluded their investigation and indicted Wang under the Narcotics Hazard Prevention Act. The indictment charges him with possessing a Category 2 narcotic, which prosecutors say is supported by a lab report. Wang's lawyer said he would answer the charge in court.",
				Development: "Prosecutors indict Wang under the Narcotics Hazard Prevention Act",
				HappenedAgo: ago(95 * day),
			},
			{
				Key: "drug-3-storm", Outlet: "storm", Ago: 94 * day,
				Headline: "【範例】從臨檢到起訴：王男毒品案的三個月",
				Summary:  "This explainer retraces the case from the roadside check through the referral to the indictment, and sets out the statutory penalties for possessing a Category 2 narcotic. It quotes an unnamed legal professional on typical sentences in similar cases and reports no new development.",
			},
			{
				Key: "drug-4-tvbs", Outlet: "tvbs", Ago: 5 * time.Hour,
				Headline:    "【範例】王男毒品案今首度開庭 當庭否認持有",
				Summary:     "The article describes today's first hearing at the district court, where Wang denied possessing drugs and said the items had been left with him by a friend. It notes that he was indicted earlier and quotes the prosecutor arguing that the test results are clear. The judge set another hearing for next month.",
				Development: "The district court holds its first hearing; Wang denies possession",
				HappenedAgo: ago(5 * time.Hour),
			},
			{
				Key: "drug-4-setn", Outlet: "setn", Ago: 4 * time.Hour,
				Headline: "【範例】毒品案開庭 王男戴口罩低頭快步進法院",
				Summary:  "The article focuses on Wang's arrival at court, describing him wearing a mask and not answering reporters' questions. It cites the hearing, in which Wang denied the charge, and notes that the judge plans to continue next month.",
				StepOf:   "drug-4-tvbs",
			},
			{
				Key: "drug-4-pts", Outlet: "pts", Ago: 3 * time.Hour,
				Headline: "【範例】王姓男子毒品案開庭 辯方質疑搜索程序",
				Summary:  "The report says the defence lawyer questioned in court whether the original search of the car was lawful, while prosecutors argued the roadside check had a legal basis. It quotes both sides and says the judge will first examine the dispute over the search.",
				StepOf:   "drug-4-tvbs",
			},
		},
	},
	{
		// A wire story run by several outlets, and a step with an unclear date.
		Title: "[Sample] Sample City Council reviews next year's budget",
		Articles: []sampleArticle{
			{
				Key: "budget-1-cna", Outlet: "cna", Ago: 2 * day,
				Headline:    "【範例】範例市府送出明年度總預算案 規模創新高",
				Summary:     "The report says the Sample City government has sent next year's budget bill to the council, with the highest spending on record. The city says the increase goes mainly to social welfare and public transport. It quotes the mayor calling the budget \"pragmatic and steady\".",
				Development: "The city government sends next year's budget bill to the council",
				HappenedAgo: ago(2 * day),
			},
			{Key: "budget-1-udn", Outlet: "udn", Ago: 2*day - 1*time.Hour, Headline: "【範例】範例市府送出明年度總預算案 規模創新高", ReprintOf: "budget-1-cna"},
			{Key: "budget-1-ltn", Outlet: "ltn", Ago: 2*day - 2*time.Hour, Headline: "【範例】範例市總預算案送議會 歲出規模創新高", ReprintOf: "budget-1-cna"},
			{Key: "budget-1-nownews", Outlet: "nownews", Ago: 2*day - 2*time.Hour, Headline: "【範例】範例市府送出明年度總預算案 規模創新高", ReprintOf: "budget-1-cna"},
			{
				Key: "budget-2-chinatimes", Outlet: "chinatimes", Ago: 26 * time.Hour,
				Headline:    "【範例】預算案舉債增加 議員批市府財政紀律鬆散",
				Summary:     "The article is built around councillors' criticism, saying the budget raises borrowing compared with this year. It quotes one councillor calling the city's fiscal discipline \"lax\" and says several councillors have recently said they will propose cuts, without giving a date. The city responds that debt remains within the legal limit.",
				Development: "Several councillors say they will propose cuts to the budget",
				Approximate: true,
			},
			{
				Key: "budget-2-newtalk", Outlet: "newtalk", Ago: 25 * time.Hour,
				Headline: "【範例】議員揚言刪預算 市府：福利支出不能等",
				Summary:  "The article opens with the city spokesperson stressing the need for the welfare and transport spending, then reports that some councillors want cuts. It says the city is willing to explain the bill item by item at the council and quotes a councillor who supports it.",
				StepOf:   "budget-2-chinatimes",
			},
			{
				Key: "budget-3-upmedia", Outlet: "upmedia", Ago: 9 * time.Hour,
				Headline:    "【範例】範例市議會開始審查總預算 首日進度緩慢",
				Summary:     "The report says the council began reviewing the budget today and finished questioning only two departments on the first day. It describes disagreement between parties over the order of review, and says the speaker announced the review will continue tomorrow.",
				Development: "The council begins reviewing the budget bill",
				HappenedAgo: ago(9 * time.Hour),
			},
			{
				Key: "budget-3-cw", Outlet: "cw", Ago: 7 * time.Hour,
				Headline: "【範例】一張圖看懂範例市明年度預算花在哪",
				Summary:  "This explainer uses charts to break down the share of each spending item in the budget and compares the past five years. It reports no new progress in the review; all figures come from the budget documents the city published.",
			},
		},
	},
	{
		// A fast-moving event covered by many outlets on the same day.
		Title: "[Sample] Typhoon Sample approaches; local governments prepare",
		Articles: []sampleArticle{
			{
				Key: "typhoon-1-cna", Outlet: "cna", Ago: 30 * time.Hour,
				Headline:    "【範例】氣象署發布颱風「範例」海上警報",
				Summary:     "The report says the weather agency has issued a sea warning for Typhoon Sample and expects its storm circle to approach waters off the east coast tomorrow. It quotes a forecaster saying the track is still uncertain and urging vessels to take care.",
				Development: "The weather agency issues a sea typhoon warning",
				HappenedAgo: ago(30 * time.Hour),
			},
			{Key: "typhoon-1-ebc", Outlet: "ebc", Ago: 29 * time.Hour, Headline: "【範例】氣象署發布颱風「範例」海上警報", ReprintOf: "typhoon-1-cna"},
			{
				Key: "typhoon-1-ftv", Outlet: "ftv", Ago: 29 * time.Hour,
				Headline: "【範例】颱風「範例」來勢洶洶 東部民眾搶購民生物資",
				Summary:  "The article centres on crowds at supermarkets on the east coast, describing shelves of vegetables and instant noodles being emptied. It quotes shoppers worried about power cuts and notes that a sea warning has been issued.",
				StepOf:   "typhoon-1-cna",
			},
			{
				Key: "typhoon-2-tvbs", Outlet: "tvbs", Ago: 12 * time.Hour,
				Headline:    "【範例】颱風「範例」陸上警報發布 範例縣列警戒區",
				Summary:     "The report says the weather agency has issued a land warning, with Sample County among the areas on alert. It quotes a forecaster saying the typhoon has strengthened slightly and will be closest tomorrow morning, and warns of heavy rain in the mountains.",
				Development: "The weather agency issues a land typhoon warning",
				HappenedAgo: ago(12 * time.Hour),
			},
			{
				Key: "typhoon-2-cti", Outlet: "cti", Ago: 11 * time.Hour,
				Headline: "【範例】陸警發布 範例縣長視察抽水站：全力戒備",
				Summary:  "The article follows the Sample County magistrate inspecting pumping stations and low-lying areas, quoting him saying the county is \"on full alert\". It notes that the land warning has been issued and lists the number of shelters the county has opened.",
				StepOf:   "typhoon-2-tvbs",
			},
			{
				Key: "typhoon-2-mirror", Outlet: "mirrormedia", Ago: 10 * time.Hour,
				Headline: "【範例】颱風前夕的漁港：船長們怎麼判斷要不要進港",
				Summary:  "This feature interviews several captains at Sample fishing port about how they decide, from experience, when to return to harbour. It reports nothing new about the typhoon and dwells on the trade-off fishers make between forecasts and income.",
			},
			{
				Key: "typhoon-3-thenewslens", Outlet: "thenewslens", Ago: 2 * time.Hour,
				Headline:    "【範例】範例縣宣布明天停止上班上課",
				Summary:     "The report says the Sample County government has announced that work and school are cancelled tomorrow because forecast winds meet the threshold for closures. It also lists neighbouring counties that have not yet announced, and the time the county made its announcement.",
				Development: "Sample County announces work and school closures",
				HappenedAgo: ago(2 * time.Hour),
			},
			{
				Key: "typhoon-3-twreporter", Outlet: "twreporter", Ago: 1 * time.Hour,
				Headline: "【範例】停班停課怎麼決定？標準與爭議一次看",
				Summary:  "After Sample County's closure announcement, the article explains the current wind and rainfall thresholds and the disputes that past decisions caused. It quotes a scholar arguing that the criteria should include traffic conditions, and the county saying it followed the rules.",
				StepOf:   "typhoon-3-thenewslens",
			},
		},
	},
}
