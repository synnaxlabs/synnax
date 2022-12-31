// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select as CoreSelect } from "./Select";
import { SelectMultiple } from "./SelectMultiple";

export type { SelectMultipleProps } from "./SelectMultiple";
export type { SelectProps } from "./Select";

type CoreSelectType = typeof CoreSelect;

interface SelectType extends CoreSelectType {
  Multiple: typeof SelectMultiple;
}

export const Select = CoreSelect as SelectType;

Select.Multiple = SelectMultiple;
