// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type ranger } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Access, Log } from "@synnaxlabs/pluto";
import { deep, type telem } from "@synnaxlabs/x";
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
  // The range the user picked custom over, or entered through the custom input. Custom
  // stays picked while the stored range is that one, even when it matches a rolling
  // preset, and lets go once an undo or another user changes it.
  const [customOver, setCustomOver] = useState<ranger.Ref | null>(null);
  const customPicked = customOver != null && deep.equal(customOver, range);
  const selected = customPicked ? PlatformRange.CUSTOM_KEY : matched;
  const handleRangeChange = useCallback(
    (itemKey: string) => {
      if (itemKey === PlatformRange.CUSTOM_KEY) {
        setCustomOver(range);
        return;
      }
      setCustomOver(null);
      const entry = entries.find((e) => e.key === itemKey);
      if (entry != null) dispatch(log.setRange({ range: PlatformRange.toRef(entry) }));
    },
    [dispatch, entries, range],
  );
  const handleWindowChange = useCallback(
    (window: telem.Window) => {
      const next: ranger.Ref = { variant: "window", window };
      setCustomOver(next);
      dispatch(log.setRange({ range: next }));
    },
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
