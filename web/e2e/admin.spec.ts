import { expect, test } from "@playwright/test";
import { findEventLink } from "./helpers";

const token = process.env.E2E_ADMIN_TOKEN;
const SAMPLE_EVENT = "[Sample] Typhoon Sample approaches; local governments prepare";

test.describe("admin", () => {
  test.skip(!token, "E2E_ADMIN_TOKEN is not set");

  test("requires signing in, and rejects a wrong token", async ({ page }) => {
    await page.goto("/admin");
    await expect(page).toHaveURL(/\/admin\/login$/);

    await page.getByLabel("Admin token").fill("not-the-token");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("alert")).toContainText("not accepted");
  });

  test("moves an article to a new event, then merges it back", async ({ page }) => {
    await page.goto("/admin/login");
    await page.getByLabel("Admin token").fill(token ?? "");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Link decisions" })).toBeVisible();

    // Find the sample event through the public site, then open its admin page.
    await page.goto("/");
    const href = await (await findEventLink(page, /Typhoon Sample approaches/)).getAttribute(
      "href",
    );
    const originalId = href?.match(/\d+$/)?.[0];
    expect(originalId).toBeTruthy();
    await page.goto(`/admin/events/${originalId}`);
    const before = await page.getByRole("row").count();

    // Move the first article out into a new event.
    const title = `E2E moved event ${Date.now()}`;
    await page.getByRole("button", { name: "Move" }).first().click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Title for the new event").fill(title);
    await dialog.getByRole("button", { name: "Move", exact: true }).click();
    await expect(page.getByRole("status")).toContainText("moved");
    await expect(page.getByRole("heading", { level: 1, name: title })).toBeVisible();

    // Merge that new event back into the original.
    await page.goto(`/admin/events/${originalId}`);
    await expect(page.getByRole("row")).toHaveCount(before - 1);
    await page.getByRole("button", { name: "Merge another event into this one" }).click();
    await page.getByRole("dialog").getByLabel("Search events by title or id").fill(title);
    await page
      .getByRole("dialog")
      .getByRole("radio", { name: new RegExp(title) })
      .check();
    await page.getByRole("dialog").getByRole("button", { name: "Merge", exact: true }).click();
    await expect(page.getByRole("status")).toContainText("merged");
    await expect(page.getByRole("row")).toHaveCount(before);
    await expect(page.getByRole("heading", { level: 1, name: SAMPLE_EVENT })).toBeVisible();
  });
});
