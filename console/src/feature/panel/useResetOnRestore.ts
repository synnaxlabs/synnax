// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query } from "@synnaxlabs/client";
import { Panel, Synnax } from "@synnaxlabs/pluto";
import { useEffect } from "react";

/**
 * Resets the surrounding error boundary once the tab's resource exists again,
 * so a restore from any session heals a tombstoned surface without a manual
 * reset.
 */
export const useResetOnRestore = (resetErrorBoundary: () => void): void => {
  const resource = Panel.useSelectTabResource({});
  const client = Synnax.use();
  useEffect(() => {
    if (client == null) return;
    return client.ontology.onChange(resource, (result) => {
      if (query.isLive(result)) resetErrorBoundary();
    });
  }, [client, resource.type, resource.key, resetErrorBoundary]);
};
