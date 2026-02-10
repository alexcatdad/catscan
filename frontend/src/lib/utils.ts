// Utility functions for the UI.

// Format a relative time string (e.g., "2 days ago").
export function formatRelativeTime(dateString: string): string {
	const date = new Date(dateString);
	const now = new Date();
	const diff = now.getTime() - date.getTime();

	const seconds = Math.floor(diff / 1000);
	const minutes = Math.floor(seconds / 60);
	const hours = Math.floor(minutes / 60);
	const days = Math.floor(hours / 24);
	const months = Math.floor(days / 30);
	const years = Math.floor(days / 365);

	if (seconds < 60) {
		return "just now";
	} else if (minutes < 60) {
		return `${minutes}m ago`;
	} else if (hours < 24) {
		return `${hours}h ago`;
	} else if (days < 30) {
		return `${days}d ago`;
	} else if (months < 12) {
		return `${months}mo ago`;
	} else {
		return `${years}y ago`;
	}
}

// Get lifecycle badge color class.
export function getLifecycleColor(lifecycle: string): string {
	switch (lifecycle) {
		case "ongoing":
			return "text-[var(--color-success)]";
		case "maintenance":
			return "text-[var(--color-info)]";
		case "stale":
			return "text-[var(--color-warning)]";
		case "abandoned":
			return "text-[var(--color-error)]";
		default:
			return "text-[var(--color-fg-muted)]";
	}
}

// Get visibility badge color class.
export function getVisibilityColor(visibility: string): string {
	return visibility === "public" ? "text-[var(--color-info)]" : "text-[var(--color-warning)]";
}

// Get CI status dot color class.
export function getCIStatusColor(status: string): string {
	switch (status) {
		case "passing":
			return "bg-[var(--color-success)]";
		case "failing":
			return "bg-[var(--color-error)]";
		default:
			return "bg-[var(--color-fg-subtle)]";
	}
}

// Check if a repo has completeness issues.
export function getCompletenessIssues(completeness: {
	HasDescription: boolean;
	HasReadme: boolean;
	HasLicense: boolean;
}): string[] {
	const issues: string[] = [];
	if (!completeness.HasDescription) issues.push("description");
	if (!completeness.HasReadme) issues.push("README");
	if (!completeness.HasLicense) issues.push("license");
	return issues;
}
