// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useInitializerRef } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { start } from "@/app/analytics/start";
import { Analytics } from "@/platform/analytics";

/**
 * Starts analytics and provides the sink to everything below. The sink is built once
 * and never replaced, so a capture written into a callback stays live for the length of
 * the window.
 */
export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const sink = useInitializerRef(start);
  return <Analytics.Provider sink={sink.current}>{children}</Analytics.Provider>;
};
