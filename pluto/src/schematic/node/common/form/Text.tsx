// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Input } from "@synnaxlabs/charon/input";
import type { Text } from "@synnaxlabs/charon/text";
import { type ReactElement } from "react";

import { Select } from "@/select";
export const SelectTextLevel = ({
  value,
  onChange,
}: Input.Control<Text.Level>): ReactElement => (
  <Select.Text.Level value={value} onChange={onChange} />
);
