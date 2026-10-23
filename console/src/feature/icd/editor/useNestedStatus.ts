// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { type Status } from "@synnaxlabs/lyra/status";
import { deep } from "@synnaxlabs/x";
import { useCallback, useSyncExternalStore } from "react";

/**
 * @returns the first error or warning on a form path under the prefix, or null. Use it
 * for a grid, whose cells are not form fields and so show no status of their own.
 */
export const useNestedStatus = (
  prefix: string,
  match?: (key: string) => boolean,
): Status.Crude | null => {
  const { bind, getStatuses } = Form.useContext();
  return useSyncExternalStore(
    bind,
    useCallback(
      () =>
        getStatuses().find(
          ({ key }) =>
            key != null && deep.pathsMatch(key, prefix) && (match?.(key) ?? true),
        ) ?? null,
      [getStatuses, prefix, match],
    ),
  );
};
