import { test, expect } from "@playwright/test";

const TEST_QUERY = "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE";

// Fail all tests if credentials are not available.
test.beforeEach(async () => {
  if (!process.env.PKB_GOOGLE_CLIENT_ID) {
    throw new Error("FAIL: PKB_GOOGLE_CLIENT_ID is not set");
  }
});

test("web UI loads with search form and source checkboxes", async ({
  page,
}) => {
  await page.goto("/");
  await expect(page.locator("input#query")).toBeVisible();
  await expect(page.locator('input[value="gdrive"]')).toBeVisible();
  await expect(page.locator('input[value="gmail"]')).toBeVisible();
});

test("search returns results from Google Drive", async ({ page }) => {
  await page.goto("/");

  // Enable gdrive, disable gmail
  const gdriveCheckbox = page.locator('input[value="gdrive"]');
  const gmailCheckbox = page.locator('input[value="gmail"]');
  if (!(await gdriveCheckbox.isChecked())) await gdriveCheckbox.check();
  if (await gmailCheckbox.isChecked()) await gmailCheckbox.uncheck();

  await page.fill("input#query", TEST_QUERY);
  await page.click('#searchForm button[type="submit"]');

  // Wait for results to appear (API call may take a few seconds)
  const resultsList = page.locator("#results li");
  await expect(resultsList.first()).toBeVisible({ timeout: 30_000 });

  // Verify at least one result mentions the test page
  const resultsText = await page.locator("#results").textContent();
  expect(resultsText).toContain("PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE");
  expect(resultsText).toContain("google-drive");
});

test("search returns results from Gmail", async ({ page }) => {
  await page.goto("/");

  // Enable gmail, disable gdrive
  const gdriveCheckbox = page.locator('input[value="gdrive"]');
  const gmailCheckbox = page.locator('input[value="gmail"]');
  if (await gdriveCheckbox.isChecked()) await gdriveCheckbox.uncheck();
  if (!(await gmailCheckbox.isChecked())) await gmailCheckbox.check();

  await page.fill("input#query", TEST_QUERY);
  await page.click('#searchForm button[type="submit"]');

  const resultsList = page.locator("#results li");
  await expect(resultsList.first()).toBeVisible({ timeout: 30_000 });

  const resultsText = await page.locator("#results").textContent();
  expect(resultsText).toContain("PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE");
  expect(resultsText).toContain("gmail");
});

test("search with both sources returns results from both", async ({
  page,
}) => {
  await page.goto("/");

  // Enable both sources
  const gdriveCheckbox = page.locator('input[value="gdrive"]');
  const gmailCheckbox = page.locator('input[value="gmail"]');
  if (!(await gdriveCheckbox.isChecked())) await gdriveCheckbox.check();
  if (!(await gmailCheckbox.isChecked())) await gmailCheckbox.check();

  await page.fill("input#query", TEST_QUERY);
  await page.click('#searchForm button[type="submit"]');

  const resultsList = page.locator("#results li");
  await expect(resultsList.first()).toBeVisible({ timeout: 30_000 });

  const resultsText = await page.locator("#results").textContent();
  expect(resultsText).toContain("google-drive");
  expect(resultsText).toContain("gmail");
});
