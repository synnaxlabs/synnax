// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement, useEffect } from "react";

import { Aether } from "@/aether";
import { flux } from "@/flux/aether";
import { type status } from "@/status/aether";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";

export interface ProviderProps extends PropsWithChildren {
  /**
   * Overrides error reporting for starting change streaming. Defaults to the
   * status aggregator.
   */
  handleError?: status.ErrorHandler;
}

/**
 * Starts change streaming on the connected client's cache, on both the main
 * thread and the aether worker.
 */
export const Provider = ({
  children,
  handleError: handleErrorOverride,
}: ProviderProps): ReactElement | null => {
  const client = Synnax.use();
  const aggregatorHandleError = Status.useErrorHandler();
  const handleError = handleErrorOverride ?? aggregatorHandleError;
  const { path } = Aether.useLifecycle({
    type: flux.PROVIDER_TYPE,
    schema: flux.providerStateZ,
    initialState: {},
  });
  useEffect(() => {
    if (client == null) return;
    handleError(
      async () => await client.cache.ensureStreaming(),
      "failed to start flux change streaming",
    );
  }, [client, handleError]);
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};
