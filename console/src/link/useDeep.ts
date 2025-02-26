// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Status, useAsyncEffect } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { getCurrent, onOpenUrl } from "@tauri-apps/plugin-deep-link";
import { useDispatch, useStore } from "react-redux";

import { Layout } from "@/layout";
import { type ClusterHandler, type Handler, PREFIX } from "@/link/types";
import { type RootState } from "@/store";

const BASE_LINK = `${PREFIX}<cluster-key>`;

const INCORRECT_FORMAT_ERROR = new Error(
  `Links must be of the form ${BASE_LINK} or ${BASE_LINK}/<resource>/<resource-key>`,
);

export const useDeep = (
  clusterHandler: ClusterHandler,
  handlers: Record<string, Handler>,
): void => {
  const handleException = Status.useExceptionHandler();
  const dispatch = useDispatch();
  const placeLayout = Layout.usePlacer();
  const store = useStore<RootState>();
  const urlHandler = async (urls: string[]) => {
    try {
      dispatch(Drift.focusWindow({}));

      // Processing URL, making sure is has valid form
      if (urls.length === 0 || !urls[0].startsWith(PREFIX))
        throw INCORRECT_FORMAT_ERROR;
      const urlParts = urls[0].slice(PREFIX.length).split("/");
      if (urlParts.length !== 1 && urlParts.length !== 3) throw INCORRECT_FORMAT_ERROR;

      const clusterKey = urlParts[0];
      const client = await clusterHandler({ store, key: clusterKey });
      if (urlParts.length === 1) return;

      const resource = urlParts[1];
      const resourceKey = urlParts[2];
      const handle = handlers[resource];
      if (handle == null)
        throw new Error(`Resource type "${resource}" is unknown to Synnax`);
      await handle({ client, dispatch, key: resourceKey, placeLayout });
    } catch (e) {
      handleException(
        e,
        `Failed to open ${(strings.naturalLanguageJoin(urls), "link")}`,
      );
    }
  };

  // Handles the case where the app is opened from a link
  useAsyncEffect(async () => {
    const urls = await getCurrent();
    if (urls == null) return;
    await urlHandler(urls);
  }, []);

  // Handles the case where the app is open and a link gets called
  useAsyncEffect(async () => {
    const unlisten = await onOpenUrl((urls) => void urlHandler(urls));
    return unlisten;
  }, []);
};
