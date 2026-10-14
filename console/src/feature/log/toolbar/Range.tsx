// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Access, Log } from "@synnaxlabs/pluto";
import { type telem } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { Range as PlatformRange } from "@/platform/range";
import { Session } from "@/session";

export const Range = (): ReactElement => {
  const key = Log.useKey();
  const range = Log.useRange();
  const dispatch = Log.useSingleDispatch();
  const hasEditPermission = Access.useUpdateGranted(log.ontologyID(key));
  const entries = Session.Range.useSelectMultiple();
  const matched = PlatformRange.useSelectKey(range);
  // Custom stays picked while the user types, even when the window they enter matches
  // a rolling preset.
  const [customPicked, setCustomPicked] = useState(false);
  const selected = customPicked ? PlatformRange.CUSTOM_KEY : matched;
  const handleRangeChange = useCallback(
    (itemKey: string) => {
      setCustomPicked(itemKey === PlatformRange.CUSTOM_KEY);
      const entry = entries.find((e) => e.key === itemKey);
      if (entry != null) dispatch(log.setRange({ range: PlatformRange.toRef(entry) }));
    },
    [dispatch, entries],
  );
  const handleWindowChange = useCallback(
    (window: telem.Window) =>
      dispatch(log.setRange({ range: { variant: "window", window } })),
    [dispatch],
  );
  return (
    <Flex.Box y gap="small">
      <PlatformRange.SelectSingleInputItem
        value={selected}
        onChange={handleRangeChange}
        disabled={!hasEditPermission}
      />
      {selected === PlatformRange.CUSTOM_KEY && (
        <PlatformRange.CustomInput
          x={false}
          value={range.variant === "window" ? range.window : undefined}
          onChange={handleWindowChange}
          autoFocus={customPicked}
          disabled={!hasEditPermission}
        />
      )}
    </Flex.Box>
  );
};
