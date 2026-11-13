import { test, expect } from "@playwright/test";
import { history, mockChat } from "./chat-fixture.ts";

test("preserves separate drafts and restores focus after Enter and clicking Send", async ({ page }) => {
  const chat = await mockChat(page);
  await page.goto("/");
  await chat.open();
  await chat.input.fill("  Bob draft\nsecond line  ");
  await chat.open("Charlie");
  await expect(chat.input).toHaveValue("");
  await chat.input.fill("Charlie draft");
  await chat.open();
  await expect(chat.input).toHaveValue("  Bob draft\nsecond line  ");
  await chat.input.press("Enter");
  await expect(chat.input).toHaveValue("");
  await expect(chat.input).toBeFocused();
  await page.keyboard.type("Another message");
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect(chat.input).toHaveValue("");
  await expect(chat.input).toBeFocused();
  await chat.open("Charlie");
  await expect(chat.input).toHaveValue("Charlie draft");
  expect(chat.pageErrors).toEqual([]);
});

test("reuses a failed send ID across chat switches and starts a new ID after editing", async ({ page }) => {
  const chat = await mockChat(page);
  await page.goto("/");
  await chat.open();
  await page.clock.install();
  chat.state.dropNextAck = true;
  await chat.input.fill("Lost acknowledgment");
  await chat.input.press("Enter");
  await expect(page.getByRole("button", { name: "Sending...", exact: true })).toBeDisabled();
  await expect(page.locator(".message-content").filter({ hasText: /^Lost acknowledgment$/ })).toHaveCount(1);
  await chat.open("Charlie");
  await chat.input.fill("Other draft");
  await page.clock.fastForward(11000);
  await chat.open();
  await expect(page.getByRole("alert")).toContainText("No confirmation received");
  await expect(chat.input).toHaveValue("Lost acknowledgment");
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect(chat.input).toHaveValue("");
  expect(chat.state.sends[0].client_id).toBe(chat.state.sends[1].client_id);
  expect(chat.state.messages).toHaveLength(1);
  await expect(page.locator(".message-content").filter({ hasText: /^Lost acknowledgment$/ })).toHaveCount(1);

  chat.state.rejectNextSend = true;
  await chat.input.fill("Rejected draft");
  await chat.input.press("Enter");
  await expect(page.getByRole("alert")).toContainText("Test rejection");
  const rejectedId = chat.state.sends.at(-1)!.client_id;
  await chat.input.fill("Edited draft");
  await chat.input.press("Enter");
  await expect(chat.input).toHaveValue("");
  expect(chat.state.sends.at(-1)!.client_id).not.toBe(rejectedId);
  await chat.open("Charlie");
  await expect(chat.input).toHaveValue("Other draft");
});

test("loads older pages without moving the reading position and supports retry", async ({ page }) => {
  const chat = await mockChat(page, history(80));
  await page.goto("/");
  await chat.open();
  const rows = page.locator("[data-message-id]");
  await expect(rows).toHaveCount(30);
  await expect(rows.first()).toHaveAttribute("data-message-id", "51");
  const list = page.locator(".message-list");
  await list.evaluate((element) => { element.scrollTop = 0; });
  const marker = page.locator('[data-message-id="51"]');
  const top = (await marker.boundingBox())!.y;
  await page.getByRole("button", { name: "Load older messages", exact: true }).click();
  await expect(rows).toHaveCount(60);
  await expect.poll(async () => Math.abs((await marker.boundingBox())!.y - top)).toBeLessThan(3);
  chat.state.failOlderPage = true;
  await list.evaluate((element) => { element.scrollTop = 0; });
  await page.getByRole("button", { name: "Load older messages", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("Test history failure");
  await expect(rows).toHaveCount(60);
  await page.getByRole("button", { name: "Retry older messages" }).click();
  await expect(rows).toHaveCount(80);
  await expect(page.getByRole("button", { name: "Load older messages", exact: true })).toHaveCount(0);
  expect(await rows.evaluateAll((elements) => elements.map((element) => Number(element.getAttribute("data-message-id")))))
    .toEqual(history(80).map((message) => message.id));
});

test("recovers more than one page after reconnecting without losing the draft", async ({ page }) => {
  const chat = await mockChat(page, history(80));
  await page.goto("/");
  await chat.open();
  const rows = page.locator("[data-message-id]");
  await expect(rows).toHaveCount(30);
  await chat.input.fill("Keep my draft");
  await page.clock.install();
  chat.disconnect();
  await expect(page.locator(".connection-status")).toContainText("disconnected");
  chat.state.messages = history(150);
  await page.clock.fastForward(2000);
  await expect(rows).toHaveCount(100); // Previously loaded 51–80 plus new 81–150.
  expect(await rows.evaluateAll((elements) => elements.map((element) => Number(element.getAttribute("data-message-id")))))
    .toEqual(history(150).slice(50).map((message) => message.id));
  expect(chat.state.requestedBeforeIds).toEqual(expect.arrayContaining([121, 91]));
  await expect(chat.input).toHaveValue("Keep my draft");
});

test("returns to login on an authenticated API 401 and clears the stored token", async ({ page }) => {
  const chat = await mockChat(page);
  chat.state.usersStatus = 401;
  await page.goto("/");
  await page.getByRole("button", { name: "New conversation", exact: true }).click();
  await expect(page).toHaveURL("/login");
  await expect(page.getByRole("status").filter({ hasText: "Your session expired" })).toHaveText("Your session expired. Please log in again.");
  expect(await page.evaluate(() => sessionStorage.getItem("token"))).toBeNull();
});

test("stops reconnecting when WebSocket authentication is rejected", async ({ page }) => {
  const chat = await mockChat(page);
  chat.state.rejectAuthentication = true;
  await page.goto("/");
  await expect(page.getByRole("status").filter({ hasText: "Your session expired" })).toHaveText("Your session expired. Please log in again.");
  const connections = chat.state.connections;
  await page.clock.install();
  await page.clock.fastForward(30000);
  expect(chat.state.connections).toBe(connections);
  await expect(page).toHaveURL("/login");
});

test("shows incorrect login credentials as a form error, not an expired session", async ({ page }) => {
  await page.route("**/api/**", (route) => route.fulfill({
    status: 401, contentType: "text/plain", body: "Invalid email or password",
  }));
  await page.goto("/login");
  await page.getByLabel("Email", { exact: true }).fill("alice@example.com");
  await page.getByLabel("Password", { exact: true }).fill("wrong-password");
  await page.getByRole("button", { name: "Login", exact: true }).click();
  await expect(page.getByRole("alert")).toHaveText("Invalid email or password");
  await expect(page.getByText("Your session expired. Please log in again.")).toHaveCount(0);
});

test("mobile back navigation preserves the draft and keeps the composer onscreen", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const chat = await mockChat(page);
  await page.goto("/");
  await chat.open();
  await chat.input.fill("Mobile draft");
  await page.getByRole("button", { name: "Back to conversations" }).click();
  await expect(page.locator(".conversation-sidebar")).toBeVisible();
  await chat.open();
  await expect(chat.input).toHaveValue("Mobile draft");
  const form = (await page.locator(".message-input-form").boundingBox())!;
  expect(form.y + form.height).toBeLessThanOrEqual(844);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});
