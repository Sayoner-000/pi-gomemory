package installer

import (
	"testing"
)

func TestHCD014SidebarQuotaRendering(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import { renderSidebar, formatProgressBar } from "./gomemory-sidebar.ts";

const fails = [];

// Test formatProgressBar
if (formatProgressBar(1.0, 10) !== "██████████") fails.push("barra 100%");
if (formatProgressBar(0.5, 10) !== "█████░░░░░") fails.push("barra 50%");
if (formatProgressBar(0.0, 10) !== "░░░░░░░░░░") fails.push("barra 0%");

const dummyTheme = {
	fg: (color, text) => text,
	bg: (color, text) => text,
	style: (text, opts) => text,
};

// Render sin antigravity
const stateNoAgy = {
	title: "HCD-014",
	tokens: 1200,
	contextLimit: 200000,
	cost: 0.05,
	servers: [{ name: "gomemory", status: "Configured" }],
	isAntigravityProvider: false,
};
const linesNoAgy = renderSidebar(stateNoAgy, 36, dummyTheme);
const textNoAgy = linesNoAgy.join("\n");
if (textNoAgy.includes("Google AI Pro Quota")) {
	fails.push("aparece quota cuando isAntigravityProvider es false");
}

// Render con antigravity y cuota cargada
const stateAgy = {
	title: "HCD-014",
	tokens: 1200,
	contextLimit: 200000,
	cost: 0.05,
	servers: [{ name: "gomemory", status: "Configured" }],
	isAntigravityProvider: true,
	quota: {
		planLabel: "Google AI Pro",
		groups: [
			{
				displayName: "Gemini Models",
				buckets: [
					{
						displayName: "Weekly Limit",
						remainingFraction: 0.66,
						remainingPercent: 66.0,
						resetIn: "2d 8h",
					},
					{
						displayName: "Five Hour Limit",
						remainingFraction: 0.95,
						remainingPercent: 95.0,
						resetIn: "3h 50m",
					},
				],
			},
			{
				displayName: "Gemini 4 Argon",
				buckets: [
					{
						displayName: "Weekly Limit",
						remainingFraction: 0.13,
						remainingPercent: 13.0,
						resetIn: "3d 21h",
					},
				],
			},
		],
	},
};

const linesAgy = renderSidebar(stateAgy, 36, dummyTheme);
const textAgy = linesAgy.join("\n");
if (!textAgy.includes("Google AI Pro Quota")) fails.push("falta header Google AI Pro Quota");
if (!textAgy.includes("Gemini Models")) fails.push("falta grupo Gemini Models");
if (!textAgy.includes("Gemini 4 Argon")) fails.push("falta grupo Gemini 4 Argon");
if (!textAgy.includes("66%")) fails.push("falta 66%");
if (!textAgy.includes("2d 8h")) fails.push("falta reset 2d 8h");

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if out != "PASS" {
		t.Fatal(out)
	}
}
