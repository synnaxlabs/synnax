// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors, Flux, Panel } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

export interface DeletedFallbackProps extends Errors.FallbackProps {
  error: Flux.DeletedError;
}

/**
 * Wraps a fallback rendering a resource tab's deleted state. The wrapped component
 * sees only a DeletedError thrown by a resource tab; a view tab reading someone
 * else's deleted resource, and every other error, get the generic fallback.
 */
export const deletedResourceFallback = (
  Deleted: FC<DeletedFallbackProps>,
): FC<Errors.FallbackProps> => {
  const Fallback = (props: Errors.FallbackProps): ReactElement => {
    const variant = Panel.useSelectTabVariant({});
    const { error } = props;
    if (variant !== "resource" || !Flux.DeletedError.matches(error))
      return <Errors.Fallback {...props} />;
    return <Deleted {...props} error={error} />;
  };
  Fallback.displayName = "DeletedResourceFallback";
  return Fallback;
};
