// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";

import { Entry, type EntryData, type EntryProps } from "@/vis/legend/Entry";

export interface EntriesProps extends Omit<EntryProps, "entry"> {
  data: Optional<EntryData, "visible">[];
}

export const Entries = memo(
  ({
    data,
    allowVisibleChange = true,
    background = 1,
    ...rest
  }: EntriesProps): ReactElement => (
    <>
      {data
        .sort((a, b) => a.label.localeCompare(b.label))
        .map(({ key, visible = true, ...entryRest }) => (
          <Entry
            key={key}
            entry={{ key, visible, ...entryRest }}
            allowVisibleChange={allowVisibleChange}
            background={background}
            {...rest}
          />
        ))}
    </>
  ),
);

Entries.displayName = "Legend.Entries";
