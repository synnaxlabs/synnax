// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { ICD } from "@synnaxlabs/pluto";
import { type ReactElement, useSyncExternalStore } from "react";

const PATH = "config.icd";

/**
 * Picks the ICD a task reads its messages from. A new task has none. Changing it
 * removes the task's messages, since their keys belong to the old ICD.
 */
export const SelectICD = (): ReactElement => {
  const { set, bind, getStatuses, mode } = Form.useContext();
  const value = Form.useFieldValue<icd.Key>(PATH, { optional: true });
  // A form keeps no field state for an absent value, so the status that configure sets
  // on a missing ICD is only in the form's statuses.
  const status = useSyncExternalStore(bind, () =>
    getStatuses().find((s) => s.key === PATH),
  );
  const preview = mode === "preview";
  const handleChange = (next: icd.Key | null) => {
    if (next === value) return;
    set(PATH, next ?? undefined);
    set("config.messages", []);
  };
  return (
    <Input.Item
      label="ICD"
      required={!preview}
      padHelpText
      helpText={status?.message}
      status={status?.variant}
      grow
    >
      <ICD.SelectSingle
        value={value ?? undefined}
        onChange={handleChange}
        preview={preview}
        grow
      />
    </Input.Item>
  );
};
