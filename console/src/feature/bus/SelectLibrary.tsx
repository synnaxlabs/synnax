// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type library } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Library } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

/**
 * Picks the library a task reads its messages from. Changing it removes the task's
 * messages, since their keys belong to the old library.
 */
export const SelectLibrary = (): ReactElement => (
  <Form.Field<library.Key>
    path="config.library"
    label="Library"
    grow
    onChange={(next, { get, set }) => {
      if (next !== get<library.Key>("config.library").value) set("config.messages", []);
    }}
  >
    {({ value, onChange, preview }) => (
      <Library.SelectSingle
        value={value}
        onChange={onChange}
        allowNone={false}
        preview={preview}
        grow
      />
    )}
  </Form.Field>
);
