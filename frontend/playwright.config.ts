import { defineConfig, devices } from "playwright";

const PORT = 9527; // Test-specific port to avoid conflicts

export default defineConfig({
	testDir: "./e2e",
	fullyParallel: false, // Run tests sequentially for stability
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 2 : 0,
	workers: 1,
	reporter: "html",
	use: {
		baseURL: `http://localhost:${PORT}`,
		trace: "on-first-retry",
		screenshot: "only-on-failure",
	},
	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] },
		},
	],
	webServer: {
		command: "make test-server",
		url: `http://localhost:${PORT}`,
		timeout: 120000, // 2 minutes for server startup
		reuseExistingServer: !process.env.CI,
	},
});
