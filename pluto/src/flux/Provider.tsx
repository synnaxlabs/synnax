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
   * Overrides error reporting for cache listeners and change streaming.
   * Defaults to the status aggregator.
   */
  handleError?: status.ErrorHandler;
  /** Async counterpart of handleError. */
  handleAsyncError?: status.AsyncErrorHandler;
}

/**
 * Routes the connected client's cache errors to the status aggregator and
 * starts change streaming, on both the main thread and the aether worker.
 */
export const Provider = ({
  children,
  handleError: handleErrorOverride,
  handleAsyncError: handleAsyncErrorOverride,
}: ProviderProps): ReactElement | null => {
  const client = Synnax.use();
  const aggregatorHandleError = Status.useErrorHandler();
  const aggregatorHandleAsyncError = Status.useAsyncErrorHandler();
  const handleError = handleErrorOverride ?? aggregatorHandleError;
  const handleAsyncError = handleAsyncErrorOverride ?? aggregatorHandleAsyncError;
  const { path } = Aether.useLifecycle({
    type: flux.PROVIDER_TYPE,
    schema: flux.providerStateZ,
    initialState: {},
  });
  useEffect(() => {
    if (client == null) return;
    client.cache.setErrorHandlers(handleError, handleAsyncError);
    handleError(
      async () => await client.cache.ensureStreaming(),
      "failed to start flux change streaming",
    );
  }, [client, handleError, handleAsyncError]);
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};
