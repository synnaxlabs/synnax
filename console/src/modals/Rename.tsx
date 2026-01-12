// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Input, Nav } from "@synnaxlabs/pluto";
import { useState } from "react";

import { type BaseArgs, createBase, type Prompt } from "@/modals/Base";
import { ModalContentLayout } from "@/modals/layout";
import { Triggers } from "@/triggers";

export interface PromptRenameLayoutArgs extends BaseArgs<string> {
  allowEmpty?: boolean;
  initialValue?: string;
  label?: string;
}

export const RENAME_LAYOUT_TYPE = "rename";

export interface PromptRename extends Prompt<string, PromptRenameLayoutArgs> {}

export const [useRename, Rename] = createBase<string, PromptRenameLayoutArgs>(
  "Name",
  RENAME_LAYOUT_TYPE,
  ({
    value: { result, allowEmpty = false, label = "Name", initialValue },
    onFinish,
  }) => {
    const [name, setName] = useState(result ?? initialValue ?? "");
    const [error, setError] = useState<string | undefined>(undefined);
    const footer = (
      <>
        <Triggers.SaveHelpText action="Save" trigger={Triggers.SAVE} />
        <Nav.Bar.End x align="center">
          <Button.Button
            status="success"
            disabled={!allowEmpty && name.length === 0}
            variant="filled"
            onClick={() => {
              if (allowEmpty && name.length === 0) return onFinish(null);
              if (!allowEmpty && name.length === 0)
                return setError(`${label} is required`);
              return onFinish(name);
            }}
            trigger={Triggers.SAVE}
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </>
    );

    return (
      <ModalContentLayout footer={footer}>
        <Input.Item
          label={label}
          required={!allowEmpty}
          helpText={error}
          status={error != null ? "error" : "success"}
          padHelpText
        >
          <Input.Text
            autoFocus
            placeholder={label}
            level="h2"
            variant="text"
            value={name}
            onChange={setName}
            selectOnFocus
          />
        </Input.Item>
      </ModalContentLayout>
    );
  },
);
