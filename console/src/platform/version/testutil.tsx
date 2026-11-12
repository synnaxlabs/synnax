// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, type resource, type Synnax as Client } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import { type Channel } from "@tauri-apps/api/core";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { type DownloadEvent } from "@tauri-apps/plugin-updater";
import { act, cleanup } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { type Mock, type MockInstance, vi } from "vitest";

import { Modals } from "@/platform/modals";
import { Version } from "@/platform/version";
import {
  CONNECTION_PARAMS,
  createCore,
  createCoreState,
} from "@/session/core/testutil";
import { createStatusConsoleWrapper, renderWithStatus } from "@/testutil";

/** The object whose update action lets a user take the Console past its Core. */
export const RESOURCE_ID: resource.ID = { type: "builtin", key: "version" };

/** The manifest URL of the Core that {@link createWrapper} selects. */
export const ENDPOINT = `http://${CONNECTION_PARAMS.host}:${CONNECTION_PARAMS.port}/console/update/latest.json`;

export interface MockUpdaterParams {
  /** The version of the running Console. A 0.0.x version is a dev build. */
  version?: string;
  /**
   * The release that the Core serves and that the Console may take, or null when it
   * serves none. An older release reaches the Console only over a secure connection.
   */
  served?: string | null;
  /** The newest release on the docs site, or null when none is newer. */
  latest?: string | null;
  /** Runs the download and sends its events. Resolves at once by default. */
  download?: (send: (event: DownloadEvent) => void) => Promise<void>;
}

export type IPC = Mock<(cmd: string, args?: unknown) => Promise<unknown>>;

/**
 * Fakes the Rust side of the updater: the app version, the check of the Core's
 * manifest, the check of the docs site, the download, the install, and the restart.
 * @returns the spy that receives every IPC call.
 */
export const mockUpdater = ({
  version = "1.5.0",
  served = null,
  latest = null,
  download = async () => {},
}: MockUpdaterParams = {}): IPC => {
  let rid = 0;
  const createMetadata = (next: string) => ({
    rid: ++rid,
    currentVersion: version,
    version: next,
    downgrade: migrate.semVerNewer(version, next),
    rawJson: {},
  });
  const ipc: IPC = vi.fn(async (cmd: string, args?: unknown) => {
    switch (cmd) {
      case "plugin:app|version":
        return version;
      case "updater_check": {
        const { secure } = args as { secure: boolean };
        if (served == null || served === version) return null;
        if (!secure && migrate.semVerNewer(version, served)) return null;
        return createMetadata(served);
      }
      case "plugin:updater|check":
        return latest == null ? null : createMetadata(latest);
      case "plugin:updater|download": {
        const { onEvent } = args as { onEvent: Channel<DownloadEvent> };
        await download((event) => onEvent.onmessage(event));
        return ++rid;
      }
      default:
        return null;
    }
  });
  mockIPC(ipc);
  return ipc;
};

/**
 * Unmounts the rendered trees, so a Provider releases its update through the mock, then
 * clears the mock.
 */
export const clearUpdater = async (): Promise<void> => {
  cleanup();
  await act(async () => {});
  clearMocks();
};

const CORE_RELEASE_PATH = "/releases/core/latest.json";

/**
 * Answers the request of the Console for the newest Core release with latest, or fails
 * it when latest is an error. Every other request goes to the network.
 * @returns the spy on fetch.
 */
export const mockLatestCore = (latest: string | Error): MockInstance<typeof fetch> => {
  const original = globalThis.fetch;
  return vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    if (!urlOf(input).endsWith(CORE_RELEASE_PATH)) return await original(input, init);
    if (latest instanceof Error) throw latest;
    return Response.json({ version: latest });
  });
};

const urlOf = (input: Parameters<typeof fetch>[0]): string =>
  input instanceof Request ? input.url : String(input);

/** Reports whether the Console requested the newest Core release through fetch. */
export const requestedLatestCore = (spy: MockInstance<typeof fetch>): boolean =>
  spy.mock.calls.some(([input]) => urlOf(input).endsWith(CORE_RELEASE_PATH));

/** Returns the calls that ipc received for the command. */
export const callsOf = (ipc: IPC, cmd: string): unknown[][] =>
  ipc.mock.calls.filter(([c]) => c === cmd);

/** A status of a connected Core at version. */
export const createStatus = (version: string): connection.Status => ({
  ...connection.DEFAULT_STATUS,
  variant: "success",
  message: "Connected",
  details: {
    ...connection.DEFAULT_STATUS.details,
    authenticated: true,
    nodeVersion: version,
    clientServerCompatible: true,
  },
});

export interface CreateWrapperParams {
  /** The version that the selected Core reports, or null before a check reports. */
  coreVersion?: string | null;
  /** The client that access checks use. */
  client?: Client | null;
  /** The middleware that the Provider wraps each install in. */
  installMiddleware?: Version.InstallMiddleware;
  /** True when the Console connects to the selected Core over TLS. */
  secure?: boolean;
}

/**
 * Creates a wrapper with a selected Core that reports coreVersion, a version Provider,
 * and a modal stack after its children.
 */
export const createWrapper = async ({
  coreVersion = "1.5.0",
  client = null,
  installMiddleware,
  secure = false,
}: CreateWrapperParams = {}): Promise<FC<PropsWithChildren>> => {
  const core = createCore("Local", { secure });
  const { wrapper: Console } = await createStatusConsoleWrapper({
    client,
    status: coreVersion == null ? undefined : createStatus(coreVersion),
    preloadedState: createCoreState([core], core.key),
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Version.Provider installMiddleware={installMiddleware}>
        {children}
        <Modals.Stack />
      </Version.Provider>
    </Console>
  );
  return Wrapper;
};

/**
 * Renders ui in a Provider with a selected Core that reports 1.5.0.
 * @returns a function that sets the status that the Core reports.
 */
export const renderProvider = async (
  ui: ReactElement,
  client: Client | null = null,
): Promise<(status: connection.Status) => void> => {
  const core = createCore("Local");
  const { setStatus } = await renderWithStatus(
    <Version.Provider>{ui}</Version.Provider>,
    {
      client,
      status: createStatus("1.5.0"),
      preloadedState: createCoreState([core], core.key),
    },
  );
  return setStatus;
};
