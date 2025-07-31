// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer } from "@synnaxlabs/client";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
} from "react";

import { Aether } from "@/aether";
import { flux } from "@/flux/aether";
import { type ListenerAdder } from "@/flux/aether/types";
import { Context } from "@/flux/Context";
import { useAsyncEffect } from "@/hooks";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

export interface ProviderProps extends PropsWithChildren {
  openStreamer?: framer.StreamOpener;
}

export const Provider = ({
  children,
  openStreamer: propsOpenStreamer,
}: ProviderProps): ReactElement => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const streamerRef = useRef<flux.Streamer>(new flux.Streamer(handleError));

  const { path } = Aether.useLifecycle({
    type: flux.Provider.TYPE,
    schema: flux.providerStateZ,
    initialState: {},
  });

  const openStreamer = useMemo(
    () => propsOpenStreamer ?? client?.openStreamer.bind(client),
    [client, propsOpenStreamer],
  );

  useAsyncEffect(async () => {
    if (openStreamer == null) return;
    await streamerRef.current.updateStreamer(openStreamer);
    return streamerRef.current.close.bind(streamerRef.current);
  }, [openStreamer]);

  const addListener: ListenerAdder = useCallback(
    ({ channel, handler, onOpen }) =>
      streamerRef.current.addListener(handler, channel, onOpen),
    [],
  );
  return (
    <Aether.Composite path={path}>
      <Context value={addListener}>{children}</Context>
    </Aether.Composite>
  );
};
