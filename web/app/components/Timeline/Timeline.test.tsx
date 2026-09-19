import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";
import type { TimelineStep } from "~/lib/api.server";
import { Timeline } from "./Timeline";

const outlet = (id: number, name: string) => ({ id, slug: `o${id}`, name });

function step(n: number, overrides: Partial<TimelineStep> = {}): TimelineStep {
  return {
    articleId: n,
    development: `Development ${n}`,
    happenedOn: `2026-03-${String(n).padStart(2, "0")}`,
    dateIsApproximate: false,
    reports: [
      { articleId: n, outlet: outlet(n, `Outlet ${n}`), publishedAt: "2026-03-01T00:00:00Z" },
    ],
    ...overrides,
  };
}

describe("Timeline", () => {
  it("renders steps in order, each linking its outlets to their article cards", () => {
    render(<Timeline steps={[step(1), step(2)]} />);
    const items = screen.getAllByText(/^Development/);
    expect(items.map((el) => el.textContent)).toEqual(["Development 1", "Development 2"]);
    expect(screen.getByRole("link", { name: "Outlet 2" })).toHaveAttribute("href", "#article-2");
  });

  it("marks an approximate date in text, not by appearance alone", () => {
    render(<Timeline steps={[step(1, { dateIsApproximate: true })]} />);
    expect(screen.getByText(/^c\. 1 Mar 2026$/)).toBeInTheDocument();
  });

  it("calls out a long quiet gap between steps", () => {
    render(<Timeline steps={[step(1), step(2, { happenedOn: "2026-06-05" })]} />);
    expect(screen.getByText("3 months later")).toBeInTheDocument();
  });

  it("collapses earlier steps when the timeline is long", () => {
    const steps = Array.from({ length: 10 }, (_, i) => step(i + 1));
    render(<Timeline steps={steps} />);
    expect(screen.getByRole("button", { name: /Show 5 earlier developments/ })).toBeInTheDocument();
    expect(screen.queryByText("Development 1")).not.toBeInTheDocument();
    expect(screen.getByText("Development 10")).toBeInTheDocument();
  });

  it("says so when there are no steps", () => {
    render(<Timeline steps={[]} />);
    expect(screen.getByText(/No developments/)).toBeInTheDocument();
  });

  it("has no detectable accessibility violations", async () => {
    const { container } = render(<Timeline steps={[step(1), step(2)]} />);
    const results = await axe(container);
    expect(results.violations).toEqual([]);
    expect(within(container).getAllByRole("list").length).toBeGreaterThan(0);
  });
});
