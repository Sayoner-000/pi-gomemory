import { readFile, realpath } from "node:fs/promises";
import { isAbsolute, relative, resolve, sep } from "node:path";
import { Type } from "@earendil-works/pi-ai";
import {
  createAgentSession, createExtensionRuntime, defineTool, ModelRuntime,
  SessionManager, SettingsManager, type ExtensionAPI, type ResourceLoader,
} from "@earendil-works/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  pi.registerTool(defineTool({
    name: "splash_review",
    label: "Splash",
    description: "Run an independent read-only Splash review. Pass only the document path " +
      "and optional review lens, never conversation history. Does not approve HCD.",
    parameters: Type.Object({
      document: Type.String({ description: "Document path relative to the repository" }),
      lens: Type.Optional(Type.String({ description: "Optional review focus" })),
    }),
    async execute(_id, params, signal, _update, ctx) {
      const document = resolve(ctx.cwd, params.document);
      // Confine to the repository, resolving symlinks, before reading anything.
      const root = await realpath(ctx.cwd);
      const rel = relative(root, await realpath(document));
      if (rel === ".." || rel.startsWith(".." + sep) || isAbsolute(rel)) {
        throw new Error(`Splash: '${params.document}' queda fuera del repositorio`);
      }
      // Fail before contacting the provider when the requested artifact is missing.
      await readFile(document, "utf8");
      const prompt = await readFile(new URL("./gomemory-splash-prompt.md", import.meta.url), "utf8");
      if (!ctx.model) throw new Error("Splash requires a selected model");
      const resourceLoader: ResourceLoader = {
        getExtensions: () => ({ extensions: [], errors: [], runtime: createExtensionRuntime() }),
        getSkills: () => ({ skills: [], diagnostics: [] }),
        getPrompts: () => ({ prompts: [], diagnostics: [] }),
        getThemes: () => ({ themes: [], diagnostics: [] }),
        getAgentsFiles: () => ({ agentsFiles: [] }),
        getSystemPrompt: () => prompt,
        getSystemPromptSource: () => undefined,
        getAppendSystemPrompt: () => [],
        getAppendSystemPromptSources: () => [],
        extendResources: () => {},
        reload: async () => {},
      };
      const modelRuntime = await ModelRuntime.create();
      const { session } = await createAgentSession({
        cwd: ctx.cwd, model: ctx.model, modelRuntime, resourceLoader,
        tools: ["read", "grep", "find", "ls"],
        excludeTools: ["mcp__*"],
        sessionManager: SessionManager.inMemory(ctx.cwd),
        settingsManager: SettingsManager.inMemory({ compaction: { enabled: false } }),
      });
      let output = "";
      let failure = "";
      const unsubscribe = session.subscribe((event) => {
        if (event.type === "message_update" && event.assistantMessageEvent.type === "text_delta") {
          output += event.assistantMessageEvent.delta;
        }
        if (event.type === "message_end" && event.message.role === "assistant" &&
            (event.message.stopReason === "error" || event.message.stopReason === "aborted")) {
          failure = event.message.errorMessage || event.message.stopReason;
        }
      });
      const abort = () => { void session.abort(); };
      let timedOut = false;
      const timer = setTimeout(() => { timedOut = true; abort(); }, 300_000);
      signal?.addEventListener("abort", abort, { once: true });
      try {
        if (signal?.aborted) throw new Error("Splash cancelled");
        await session.prompt(JSON.stringify({ document, repository: ctx.cwd, lens: params.lens }));
        if (timedOut || signal?.aborted) throw new Error("Splash cancelled or timed out");
        if (failure || !output.trim()) throw new Error(failure || "Splash returned no review");
        return { content: [{ type: "text", text: output }], details: { document, independent: true } };
      } finally {
        clearTimeout(timer);
        signal?.removeEventListener("abort", abort);
        unsubscribe();
        session.dispose();
      }
    },
  }));
}
