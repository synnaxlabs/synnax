import { useAsyncEffect } from "@synnaxlabs/pluto";
import { resourceDir } from "@tauri-apps/api/path";
import { Command } from "@tauri-apps/plugin-shell";
import { useState } from "react";
import {
  type RAL,
  ReadableStreamMessageReader,
  WriteableStreamMessageWriter,
} from "vscode-jsonrpc";
import { type MessageTransports } from "vscode-languageclient/lib/common/client";

export const useLSP = (): MessageTransports | undefined => {
  const [messageTransports, setMessageTransports] = useState<
    MessageTransports | undefined
  >(undefined);

  useAsyncEffect(async () => {
    try {
      const resDir = await resourceDir();
      const lspPath = `${resDir}/binaries/lua-language-server/bin`;

      // Create the command to run the Lua language server
      const command = Command.create("lua-language-server", [], { cwd: lspPath });
      const child = await command.spawn();

      // Create an adapter that maps Tauri's EventEmitter to the stream interface expected by ReadableStreamMessageReader
      const stdoutAdapter: RAL.ReadableStream = {
        onData: (listener: (data: Uint8Array) => void) => {
          // Wrap the listener to convert string to Uint8Array
          const wrappedListener = (data: string) => {
            console.log(data);
            const encoded = new TextEncoder().encode(data);
            listener(encoded);
          };
          // Subscribe to the stdout "data" event
          command.stdout.on("data", wrappedListener);
          // Return a disposable object
          return {
            dispose: () => command.stdout.off("data", wrappedListener),
          };
        },
        onClose: (listener: () => void) => {},
        onError: (listener: (error: any) => void) => {},
        onEnd: (listener: () => void) => {},
      };

      // Use the adapter with vscode-jsonrpc's reader
      const reader = new ReadableStreamMessageReader(stdoutAdapter);
      // Adapter for writing: wraps child.write so it fits the _WritableStream interface.
      const writableAdapter: RAL.WritableStream = {
        write: (data: string | Uint8Array) => {
          //   try {
          //     console.log("CDC", new TextDecoder().decode(data));
          //   } catch (error) {
          //     console.error(error);
          //   }
          //   void child.write(data);
        },
        onEnd: (listener: () => void) => ({ dispose: () => {} }),
        end: () => {},
        onError: (listener: (error: Error) => void) =>
          // If your process supports error events, attach them here.
          // For now, we return a stub disposable.
          ({ dispose: () => {} }),
        onClose: (listener: () => void) =>
          // Similarly, attach to a close event if available.
          ({ dispose: () => {} }),
      };

      const writer = new WriteableStreamMessageWriter(writableAdapter);

      // Set the transports for use in your JSON-RPC connection
      setMessageTransports({ reader, writer });

      // Cleanup when the component unmounts
      return () => {
        reader.dispose();
        writer.dispose();
        void child.kill();
      };
    } catch (error) {
      console.error("Failed to start Lua LSP:", error);
    }
  }, []);

  return messageTransports;
};
