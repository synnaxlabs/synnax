// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type ranger } from "@synnaxlabs/client";
import { Access, Log } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { Range as PlatformRange } from "@/platform/range";

export const Range = (): ReactElement => {
  const key = Log.useKey();
  const range = Log.useRange();
  const dispatch = Log.useSingleDispatch();
  const hasEditPermission = Access.useUpdateGranted(log.ontologyID(key));
  const handleChange = useCallback(
    (range: ranger.Ref) => dispatch(log.setRange({ range })),
    [dispatch],
  );
  return (
    <PlatformRange.SelectRefInputItem
      value={range}
      onChange={handleChange}
      disabled={!hasEditPermission}
    />
  );
};
