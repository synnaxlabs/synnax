// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";

import { type JSONType } from "@/platform/task/payload/types";

export const JSON_TYPE_ITEMS = (
  <>
    <Select.Item<JSONType> itemKey="number">Number</Select.Item>
    <Select.Item<JSONType> itemKey="string">String</Select.Item>
    <Select.Item<JSONType> itemKey="boolean">Boolean</Select.Item>
  </>
);
