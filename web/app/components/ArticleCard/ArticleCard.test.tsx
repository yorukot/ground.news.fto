import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";
import type { Article } from "~/lib/api.server";
import { ArticleCard } from "./ArticleCard";

const article: Article = {
  id: 7,
  outlet: { id: 1, slug: "cna", name: "中央社" },
  headline: "【範例】原始標題",
  url: "https://example.com/a",
  publishedAt: "2026-09-01T02:00:00Z",
  summary: "A short summary.",
  headlineOnly: false,
  reprints: [
    {
      outlet: { id: 3, slug: "udn", name: "聯合新聞網" },
      url: "https://example.com/b",
      publishedAt: "2026-09-01T03:00:00Z",
    },
  ],
};

describe("ArticleCard", () => {
  it("shows the original headline unchanged, marked as Chinese", () => {
    render(<ArticleCard article={article} />);
    const headline = screen.getByRole("heading", { name: "【範例】原始標題" });
    expect(headline).toHaveAttribute("lang", "zh-Hant-TW");
  });

  it("links to the original safely in a new tab", () => {
    render(<ArticleCard article={article} />);
    const link = screen.getByRole("link", { name: /Read the original/ });
    expect(link).toHaveAttribute("href", "https://example.com/a");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", expect.stringContaining("noopener"));
  });

  it("is the jump target for timeline chips", () => {
    const { container } = render(<ArticleCard article={article} />);
    expect(container.querySelector("#article-7")).not.toBeNull();
  });

  it("folds wire reprints behind a disclosure", () => {
    render(<ArticleCard article={article} />);
    expect(screen.getByRole("button", { name: /Also run by 1 other outlet/ })).toBeInTheDocument();
  });

  it("explains a headline-only article instead of summarizing it", () => {
    render(<ArticleCard article={{ ...article, summary: "", headlineOnly: true, reprints: [] }} />);
    expect(screen.getByText(/doesn't allow AI to read its articles/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Read the original/ })).toBeInTheDocument();
  });

  it("has no detectable accessibility violations", async () => {
    const { container } = render(<ArticleCard article={article} />);
    expect((await axe(container)).violations).toEqual([]);
  });
});
