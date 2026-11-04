// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax as Client } from "@synnaxlabs/client";
import { TEST_CLIENT_PARAMS, TEST_CREDENTIALS } from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { scheduler } from "@synnaxlabs/x";
import { emit } from "@tauri-apps/api/event";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, cleanup, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { expect, type Mock, onTestFinished, vi } from "vitest";
import { z } from "zod";

import { Embedded } from "@/feature/embedded";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  disableActEnvironment,
  type TestStore,
} from "@/testutil";

/** The state of the embedded Core that the Desktop shell reports. */
export type Status = Parameters<Parameters<typeof Embedded.onStatusChange>[0]>[0];

/** The run history of the embedded Core that the Desktop shell keeps. */
export type History = Awaited<ReturnType<typeof Embedded.retrieveHistory>>;

export const RUNNING: Status = {
  state: "running",
  connection: {
    host: "127.0.0.1",
    port: 49152,
    username: "synnax",
    password: "launch-secret",
  },
};

/** A running Core at the address of the test Core, with its credentials. */
export const LIVE: Status = {
  state: "running",
  connection: {
    host: TEST_CLIENT_PARAMS.host,
    port: Number(TEST_CLIENT_PARAMS.port),
    ...TEST_CREDENTIALS,
  },
};

/** Deletes the API keys of the client's user that the test creates, once it ends. */
export const deleteKeysCreatedByTest = async (client: Client): Promise<void> => {
  const before = new Set((await client.auth.apiKeys.retrieve()).map((k) => k.key));
  onTestFinished(async () => {
    const created = (await client.auth.apiKeys.retrieve()).filter(
      (k) => !before.has(k.key),
    );
    await client.auth.apiKeys.delete(...created.map((k) => k.key));
  });
};

/** @returns The keys of the API keys of the client's user, sorted. */
export const retrieveKeyIDs = async (client: Client): Promise<string[]> =>
  (await client.auth.apiKeys.retrieve()).map((k) => k.key).sort();

/** Creates a client of the test Core that logs in with an API key, as a script does. */
export const createScriptClient = (apiKey: string): Client => {
  const { host, port } = TEST_CLIENT_PARAMS;
  const script = new Client({ host, port: Number(port), apiKey });
  onTestFinished(() => script.close());
  return script;
};

export interface MockedSupervisor {
  /** Receives the name of every supervisor command the app invokes. */
  commands: Mock<(cmd: string) => void>;
  /**
   * Delivers a status change as the Desktop shell does, once the app listens for it.
   */
  emitStatus: (status: Status) => Promise<void>;
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
  retrieveStatus: () => Status | Promise<Status>,
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
      await act(async () => await emit("supervisor://status", status));
    },
  };
};

/** Stands in for the discovery file that the Desktop shell writes for scripts. */
export interface DiscoveryFile {
  /** The API key the app saved last, or null before the first save. */
  apiKey: string | null;
}

const saveArgsZ = z.object({ apiKey: z.string(), replaces: z.string().nullable() });

/**
 * Creates a discovery file and the supervisor command handlers that read and write it,
 * with the compare-and-swap of the shell. Pass the handlers to {@link mockSupervisor}.
 * @param beforeSave Runs before each save compares, as another window that saves first.
 */
export const createDiscoveryFile = (
  apiKey: string | null = null,
  beforeSave?: (file: DiscoveryFile, apiKey: string) => void | Promise<void>,
): { file: DiscoveryFile; handlers: Record<string, CommandHandler> } => {
  const file: DiscoveryFile = { apiKey };
  return {
    file,
    handlers: {
      supervisor_api_key: () => file.apiKey,
      supervisor_save_api_key: async (args) => {
        const { apiKey, replaces } = saveArgsZ.parse(args);
        await beforeSave?.(file, apiKey);
        if (file.apiKey !== replaces) return false;
        file.apiKey = apiKey;
        return true;
      },
    },
  };
};

/**
 * Unmounts the tree, then removes the mocked shell once its async cleanups unlisten.
 */
export const clearSupervisor = async (): Promise<void> => {
  cleanup();
  await scheduler.flushTaskQueue();
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
  disableActEnvironment();
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
