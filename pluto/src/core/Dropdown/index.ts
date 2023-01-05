// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dropdown as CoreDropdown, useDropdown } from "./Dropdown";
export type { DropdownProps } from "./Dropdown";

type CoreDropdownType = typeof CoreDropdown;

interface DropdownType extends CoreDropdownType {
  use: typeof useDropdown;
}

export const Dropdown = CoreDropdown as DropdownType;

Dropdown.use = useDropdown;
