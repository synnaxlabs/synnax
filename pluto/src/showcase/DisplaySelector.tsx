// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@synnaxlabs/x";

import { Select } from "@/select";

import { DISPLAY } from "./constants";

interface DisplaySelectorProps {
  display: (typeof DISPLAY)[number][];
  setDisplay: (display: (typeof DISPLAY)[number][]) => void;
}

export const DisplaySelector = ({ display, setDisplay }: DisplaySelectorProps) => (
  <Select.Buttons multiple keys={DISPLAY} value={display} onChange={setDisplay}>
    {DISPLAY.map((d) => (
      <Select.Button key={d} itemKey={d}>
        {caseconv.capitalize(d)}
      </Select.Button>
    ))}
  </Select.Buttons>
);
