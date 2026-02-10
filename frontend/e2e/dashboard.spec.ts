import { test, expect } from "@playwright/test";
import { checkA11y } from "./accessibility";

test.describe("Dashboard", () => {
	test.beforeEach(async ({ page }) => {
		await page.goto("/");
	});

	test("dashboard loads and displays repos", async ({ page }) => {
		// Check header renders
		await expect(page.locator("header")).toBeVisible();
		await expect(page.locator("h1")).toContainText("CatScan");

		// Check summary cards render
		await expect(page.locator('[data-testid="summary-cards"]')).toBeVisible();

		// Check repo table renders
		await expect(page.locator('[data-testid="repo-table"]')).toBeVisible();

		// Run accessibility scan
		await checkA11y(page, "dashboard loaded");
	});

	test("filtering works", async ({ page }) => {
		// Open filter menu
		await page.click('button:has-text("Filters")');

		// Apply lifecycle filter to "stale"
		await page.click('text=Stale');

		// Wait for filtered results
		await page.waitForTimeout(100);

		// Verify only stale repos are shown
		const repoNames = await page.locator('[data-testid="repo-name"]').allTextContents();
		for (const name of repoNames) {
			// In our fixtures, repo-stale is the only stale repo
			expect(name).toContain("stale");
		}

		// Run accessibility scan on filtered state
		await checkA11y(page, "stale filter applied");

		// Clear filters
		await page.click('button:has-text("Filters")');
		await page.click('text=Clear Filters');

		// Verify all repos return
		await expect(page.locator('[data-testid="repo-name"]')).toHaveCount(6);
	});

	test("sorting works", async ({ page }) => {
		// Open sort menu
		await page.click('button:has-text("Sort:")');

		// Sort by name ascending
		await page.click('text=Name');

		// Wait for sort
		await page.waitForTimeout(100);

		// Verify alphabetical order
		const names = await page.locator('[data-testid="repo-name"]').allTextContents();
		const sortedNames = [...names].sort();
		expect(names).toEqual(sortedNames);

		// Run accessibility scan
		await checkA11y(page, "sorted by name");

		// Reverse sort order
		await page.click('button:has-text("Sort:")');
		await page.click('text=Reverse');

		// Wait for sort
		await page.waitForTimeout(100);

		// Verify reversed order
		const reversedNames = await page.locator('[data-testid="repo-name"]').allTextContents();
		expect(reversedNames).toEqual([...names].reverse());
	});

	test("row expansion shows detail", async ({ page }) => {
		// Click first repo row
		await page.click('[data-testid="repo-row"]:first-child');

		// Verify detail panel expands
		await expect(page.locator('[data-testid="repo-detail"]')).toBeVisible();

		// Check for description, topics, completeness
		await expect(page.locator('[data-testid="repo-description"]')).toBeVisible();
		await expect(page.locator('[data-testid="repo-topics"]')).toBeVisible();
		await expect(page.locator('[data-testid="repo-completeness"]')).toBeVisible();

		// Run accessibility scan on expanded state
		await checkA11y(page, "row expanded");

		// Click another row
		await page.click('[data-testid="repo-row"]:nth-child(2)');

		// Wait for transition
		await page.waitForTimeout(200);

		// Verify only one detail panel is visible
		const detailCount = await page.locator('[data-testid="repo-detail"]').count();
		expect(detailCount).toBe(1);
	});

	test("config panel opens and validates", async ({ page }) => {
		// Open settings panel
		await page.click('[aria-label="Settings"]');

		// Verify settings panel is visible
		await expect(page.locator('[data-testid="settings-panel"]')).toBeVisible();

		// Verify all config fields are populated
		await expect(page.locator("#scanPath")).toHaveValue(/repos/);
		await expect(page.locator("#port")).toHaveValue("9527");

		// Run accessibility scan on settings panel
		await checkA11y(page, "settings panel open");

		// Test validation by entering an invalid port
		await page.fill("#port", "100");

		// Try to save
		await page.click('button:has-text("Save Settings")');

		// Verify inline error appears
		await expect(page.locator("text=Port must be between 1024 and 65535")).toBeVisible();

		// Fix the value
		await page.fill("#port", "8080");

		// Save
		await page.click('button:has-text("Save Settings")');

		// Wait for save
		await page.waitForTimeout(500);

		// Reopen panel and verify value persisted
		await page.click('[aria-label="Settings"]');
		await expect(page.locator("#port")).toHaveValue("8080");
	});
});
