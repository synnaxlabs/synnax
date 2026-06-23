import { type Synnax } from "@synnaxlabs/client";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { type LSPMessage, type LSPStream, useLanguageServer } from "@/code/lsp";

const { startMock, stopMock } = vi.hoisted(() => ({ startMock: vi.fn(), stopMock: vi.fn() }));
vi.mock("vscode-languageclient/browser", () => {
  class BaseLanguageClient { start = startMock; stop = stopMock; }
  return { BaseLanguageClient, CloseAction: { DoNotRestart: 1 }, ErrorAction: { Continue: 1 } };
});

describe("dbg", () => {
  it("logs error", async () => {
    startMock.mockResolvedValue(undefined);
    const stream: LSPStream = { receive: () => new Promise<LSPMessage>(() => {}), send: vi.fn(), closeSend: vi.fn() };
    const open = vi.fn().mockResolvedValue(stream);
    const onStatus = vi.fn();
    renderHook(() => useLanguageServer({ monaco: {}, client: {} as Synnax, languageID: "arc", open, onStatus }));
    await waitFor(() => expect(onStatus.mock.calls.length).toBeGreaterThan(1));
    for (const c of onStatus.mock.calls) { const s:any = c[0]; console.log("STATUS>", s.variant, "|", s.message, "|", s.description); }
  });
});
