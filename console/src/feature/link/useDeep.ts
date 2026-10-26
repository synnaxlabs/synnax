// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { strings, sync, TimeSpan } from "@synnaxlabs/x";
import { useLayoutEffect, useRef } from "react";

import { Link } from "@/platform/link";
import { Session } from "@/session";

const BASE_LINK = `${Link.PREFIX}<cluster-key>`;

const INCORRECT_FORMAT_ERROR_MESSAGE = `Links must be of the form ${BASE_LINK} or ${BASE_LINK}/<resource>/<resource-key>.`;

const SETTLE_TIMEOUT = TimeSpan.seconds(30);

// A link outlives the renders that turn its preconditions true, so callers park here
// and an effect releases them when met transitions. A timeout rejects parked calls.
const useWaitFor = (
  met: boolean,
  timeout?: TimeSpan,
  timeoutMessage?: string,
): (() => Promise<void>) => {
  const metRef = useRef(met);
  const waitersRef = useRef<(() => void)[]>([]);
  useLayoutEffect(() => {
    metRef.current = met;
    if (!met) return;
    waitersRef.current.forEach((resolve) => resolve());
    waitersRef.current = [];
  }, [met]);
  return async (): Promise<void> => {
    if (metRef.current) return;
    let release: (() => void) | undefined;
    const released = new Promise<void>((resolve) => {
      release = resolve;
      waitersRef.current.push(resolve);
    });
    if (timeout == null) return await released;
    // A release clears every waiter, so only a timeout removes its own.
    await sync.withTimeout(released, timeout, () => {
      waitersRef.current = waitersRef.current.filter((w) => w !== release);
      return new Error(timeoutMessage);
    });
  };
};

export const useDeep = (
  connect: Link.Connect,
  handlers: Record<string, Link.Handler>,
  deps: Link.Deps = Link.DEFAULT_DEPS,
): void => {
  const handleError = Status.useAsyncErrorHandler();
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  const awaitSettled = useWaitFor(
    Session.useSettled(),
    SETTLE_TIMEOUT,
    "Timed out waiting for the workspace to settle.",
  );
  const waitProject = useWaitFor(Session.Project.useSelectIsAnySelected());
  const awaitProject = async (): Promise<void> => {
    dispatch(Session.Link.beginProjectWait());
    await waitProject();
    dispatch(Session.Link.endProjectWait());
  };
  const awaitPanel = useWaitFor(Session.Panel.useSelectSelected() != null);
  // A parked link outlives the handlers and client of the render that received it.
  // Synced refs resolve both at invoke time.
  const handlersRef = useSyncedRef(handlers);
  const clientRef = useSyncedRef(Synnax.use());
  const urlHandler = async (urls: string[]) =>
    await handleError(
      async () => {
        dispatch(Drift.focusWindow({}));

        if (urls.length === 0 || !urls[0].startsWith(Link.PREFIX))
          throw new Error(INCORRECT_FORMAT_ERROR_MESSAGE);
        const urlParts = urls[0].slice(Link.PREFIX.length).split("/");
        if (urlParts.length !== 1 && urlParts.length !== 3)
          throw new Error(INCORRECT_FORMAT_ERROR_MESSAGE);

        const client = await connect(urlParts[0]);
        if (urlParts.length === 1) return;
        const coreKey = Session.Core.selectSelectedKey(store.getState());

        // A link opens only into a ready workspace.
        await awaitSettled();
        await awaitProject();
        await awaitPanel();
        // The workspace fills the panel cache lazily, so a selection the synchronizers
        // repaired can stay cold. Retrieve it before placement.
        const panelKey = Session.Panel.selectSelected(store.getState());
        if (panelKey != null) await clientRef.current?.panels.retrieve(panelKey);
        // A later link that switched Cores supersedes this one. The handler would run
        // against the new Core's session, so drop it.
        if (Session.Core.selectSelectedKey(store.getState()) !== coreKey) return;

        const resource = urlParts[1];
        const resourceKey = urlParts[2];
        const handle = handlersRef.current[resource];
        if (handle == null)
          throw new Error(`Resource type "${resource}" is unknown to Synnax.`);
        await handle({ client: clientRef.current ?? client, key: resourceKey });
      },
      {
        message: urls.length > 1 ? "Failed to open links" : "Failed to open link",
        name: strings.naturalLanguageJoin(urls),
      },
    );

  Link.useReceive(urlHandler, deps);
};
