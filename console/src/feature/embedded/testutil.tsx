// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { emit } from "@tauri-apps/api/event";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { cleanup, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { expect, type Mock, vi } from "vitest";

import { Embedded } from "@/feature/embedded";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore } from "@/testutil";

export const RUNNING: Embedded.Status = {
  state: "running",
  connection: {
    host: "127.0.0.1",
    port: 49152,
    username: "synnax",
    password: "launch-secret",
    version: "1.4.0",
  },
};

export interface MockedSupervisor {
  /** Receives the name of every supervisor command the app invokes. */
  commands: Mock<(cmd: string) => void>;
  /**
   * Delivers a status change as the Desktop shell does, once the app listens for it.
   */
  emitStatus: (status: Embedded.Status) => Promise<void>;
}

/** Answers one supervisor command from the arguments the app sent with it. */
export type CommandHandler = (args: unknown) => unknown;

/**
 * Stands in for the Desktop shell. Undo it with {@link clearSupervisor}.
 * @param retrieveStatus - Answers the status command.
 * @param handlers - Answer other commands, keyed by command name. A command without a
 * handler resolves with nothing.
 */
export const mockSupervisor = (
  retrieveStatus: () => Embedded.Status | Promise<Embedded.Status>,
  handlers: Record<string, CommandHandler> = {},
): MockedSupervisor => {
  const commands = vi.fn<(cmd: string) => void>();
  mockIPC(
    (cmd, args) => {
      if (!cmd.startsWith("supervisor_")) return undefined;
      commands(cmd);
      if (cmd === "supervisor_status") return retrieveStatus();
      return handlers[cmd]?.(args);
    },
    { shouldMockEvents: true },
  );
  return {
    commands,
    emitStatus: async (status) => {
      // The app asks for the status only after its listener is in place.
      await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_status"));
      await emit("supervisor://status", status);
    },
  };
};

/** Removes the mocked shell. The tree unmounts first, while it can still unlisten. */
export const clearSupervisor = (): void => {
  cleanup();
  clearMocks();
};

const SynnaxProvider = ({ children }: PropsWithChildren): ReactElement => (
  <Synnax.Provider connParams={Embedded.useConnParams()}>{children}</Synnax.Provider>
);

/**
 * Creates the provider stack of a Desktop window: the client follows the embedded Core,
 * exactly as the production Pluto context does, and a modal stack is mounted.
 */
export const createDesktopWrapper = async (): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
}> => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client: null });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Embedded.Provider>
      <Console>
        <SynnaxProvider>
          <Session.SettledProvider>{children}</Session.SettledProvider>
          <Modals.Stack />
        </SynnaxProvider>
      </Console>
    </Embedded.Provider>
  );
  Wrapper.displayName = "DesktopWrapper";
  return { wrapper: Wrapper, store };
};
