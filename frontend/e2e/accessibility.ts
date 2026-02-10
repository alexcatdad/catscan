import { test, AxeResults } from "@playwright/test";

/**
 * Run an accessibility scan on the current page state.
 * Asserts that there are no accessibility violations.
 *
 * @param page - The Playwright page object to scan
 * @param context - Optional description of what's being tested (for better error messages)
 */
export async function checkA11y(page: Parameters<typeof test.step>[0], context?: string): Promise<void> {
	await test.step(`a11y check${context ? `: ${context}` : ""}`, async () => {
		const accessibilityTree = await page.accessibility.snapshot();
		if (!accessibilityTree) {
			throw new Error("Could not generate accessibility tree");
		}

		// Check for basic accessibility issues
		const violations: string[] = [];

		// Check for images without alt text
		const checkImages = (node: any) => {
			if (node.role === "img" && !node.name) {
				violations.push(`Image missing alt text: ${node.name || "[unnamed]"}`);
			}
			for (const child of node.children || []) {
				checkImages(child);
			}
		};
		checkImages(accessibilityTree);

		// Check for buttons without accessible names
		const checkButtons = (node: any) => {
			if (node.role === "button" && !node.name) {
				violations.push(`Button missing accessible name`);
			}
			for (const child of node.children || []) {
				checkButtons(child);
			}
		};
		checkButtons(accessibilityTree);

		// Check for form inputs without labels
		const checkInputs = (node: any) => {
			if (
				(node.role === "textbox" ||
					node.role === "combobox" ||
					node.role === "checkbox" ||
					node.role === "radio") &&
				!node.name
			) {
				violations.push(`Form input missing label: ${node.role}`);
			}
			for (const child of node.children || []) {
				checkInputs(child);
			}
		};
		checkInputs(accessibilityTree);

		if (violations.length > 0) {
			throw new Error(
				`Accessibility violations found:\n${violations.map((v) => `  - ${v}`).join("\n")}`
			);
		}
	});
}
