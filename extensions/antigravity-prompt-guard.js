/**
 * Temporary, reversible compatibility workaround for pi-antigravity 0.14.0.
 * Upstream: https://github.com/Rahularya01/pi-antigravity/issues/152
 * Installed by pi-gomemory until upstream resolves issue #152.
 *
 * IMPORTANT: all instruction text is retained, but moving it to a user turn
 * lowers its priority. This does not guarantee equivalent prompt-injection safety.
 * Disable for a new Pi process with ANTIGRAVITY_PROMPT_GUARD=off, or remove
 * this file and /reload. No credentials, payloads, or prompts are logged.
 */
const isRecord = (value) =>
  value !== null && typeof value === "object" && !Array.isArray(value);

export const MARKER = "[Operating instructions]\n";
export const SHORT_SYSTEM = "You are a coding assistant.";

export function relocateSystemPrompt(payload, provider, enabled = true) {
  if (
    !enabled || provider !== "antigravity" || !isRecord(payload) ||
    payload.requestType !== "agent"
  ) return payload;

  const request = payload.request;
  if (
    !isRecord(request) || !isRecord(request.systemInstruction) ||
    !Array.isArray(request.contents)
  ) return payload;

  const system = request.systemInstruction;
  // Unknown instruction metadata is not silently dropped or reinterpreted.
  if (Object.keys(system).some((key) => key !== "role" && key !== "parts")) return payload;
  if (
    !Array.isArray(system.parts) || !system.parts.length ||
    system.parts.some((part) =>
      !isRecord(part) || typeof part.text !== "string" ||
      Object.keys(part).some((key) => key !== "text")
    )
  ) return payload;
  if (
    request.contents.some((turn) =>
      !isRecord(turn) || typeof turn.role !== "string" || !Array.isArray(turn.parts) ||
      turn.parts.some((part) => !isRecord(part))
    )
  ) return payload;

  const text = system.parts.map((part) => part.text).join("\n\n");
  if (!text.trim()) return payload;
  const user = request.contents.find((turn) => turn.role === "user");
  if (
    text === SHORT_SYSTEM && user?.parts.some((part) =>
      typeof part.text === "string" && part.text.startsWith(MARKER)
    )
  ) return payload;

  let output;
  try { output = structuredClone(payload); } catch { return payload; }
  const block = { text: MARKER + text + "\n\n" };
  const target = output.request.contents.find((turn) => turn.role === "user");
  if (target) target.parts.unshift(block);
  else output.request.contents.unshift({ role: "user", parts: [block] });
  output.request.systemInstruction = {
    ...output.request.systemInstruction,
    parts: [{ text: SHORT_SYSTEM }],
  };
  return output;
}

export default function register(pi) {
  pi.on("before_provider_request", (event, ctx) => {
    const original = event.payload;
    const result = relocateSystemPrompt(
      original,
      event.model?.provider ?? ctx.model?.provider,
      process.env.ANTIGRAVITY_PROMPT_GUARD !== "off",
    );
    return result === original ? undefined : result;
  });
}
