import { status, type Synnax } from "@synnaxlabs/client";
import { type BaseLanguageClient } from "vscode-languageclient/browser";
/** LSPMessage is a single framed LSP message carried over the transport. */
export interface LSPMessage {
    content: string;
}
/** LSPStream is the bidirectional message stream a language server is reached through. It
 * is structurally a freighter stream of LSPMessages, so a concrete
 * `Stream<lspMessageZ, lspMessageZ>` satisfies it without code depending on the schema. */
export interface LSPStream {
    receive: () => Promise<LSPMessage>;
    send: (req: LSPMessage) => void;
    closeSend: () => void;
}
export interface LSPClientHandle {
    client: BaseLanguageClient;
    closed: Promise<void>;
}
export interface UseLanguageServerParams {
    /** monaco gates connection: the server only starts once monaco has initialized. */
    monaco: unknown;
    /** client is the cluster connection the server stream is opened against. */
    client: Synnax | null;
    /** languageID is the document selector the client subscribes to. */
    languageID: string;
    /** open opens the LSP message stream against the cluster. */
    open: (client: Synnax) => Promise<LSPStream>;
    /** onStatus reports connection state transitions as status.Status values. */
    onStatus: (status: status.Status) => void;
}
/** useLanguageServer runs a reconnecting language client for the lifetime of the calling
 * component, opening the stream via open and reporting connection state through onStatus.
 * It is a no-op (and reports a disabled status) until both monaco and client are ready.
 * Connection failures never throw into render — they surface only through onStatus. */
export declare const useLanguageServer: ({ monaco, client, languageID, open, onStatus, }: UseLanguageServerParams) => void;
//# sourceMappingURL=lsp.d.ts.map