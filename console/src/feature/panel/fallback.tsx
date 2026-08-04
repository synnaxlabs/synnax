// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors, type Flux, Panel } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

export interface DeletedFallbackProps extends Errors.FallbackProps {
  error: Flux.DeletedError;
}

/**
 * Routes a fallback by the tab's variant. A resource tab's fallback reads its own
 * resource, so it renders only for a resource tab; a view tab that threw while
 * reading someone else's resource falls through to Other.
 */
export const resourceOnly = <P extends Errors.FallbackProps>(
  Resource: FC<P>,
  Other: FC<P> = Errors.Fallback,
): FC<P> => {
  const Fallback = (props: P): ReactElement => {
    const variant = Panel.useSelectTabVariant({});
    if (variant !== "resource") return <Other {...props} />;
    return <Resource {...props} />;
  };
  Fallback.displayName = "ResourceOnlyFallback";
  return Fallback;
};
