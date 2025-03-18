// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Input, Nav, type Triggers as PTrigger } from "@synnaxlabs/pluto";
import { useState } from "react";

import { type BaseArgs, createBase, type Prompt } from "@/modals/Base";
import { ModalContentLayout } from "@/modals/layout";
import { Triggers } from "@/triggers";

export interface PromptRenameLayoutArgs extends BaseArgs<string> {
  result?: string;
  allowEmpty?: boolean;
}

export const RENAME_LAYOUT_TYPE = "rename";

export interface PromptRename extends Prompt<string, PromptRenameLayoutArgs> {}

const SAVE_TRIGGER: PTrigger.Trigger = ["Enter"];

export const [useRename, Rename] = createBase<string, PromptRenameLayoutArgs>(
  "Name",
  RENAME_LAYOUT_TYPE,
  ({ value: { result, allowEmpty = false }, onFinish }) => {
    const [name, setName] = useState(result ?? "");
    const [error, setError] = useState<string | undefined>(undefined);

    const footer = (
      <>
        <Triggers.SaveHelpText action="Save" trigger={SAVE_TRIGGER} />
        <Nav.Bar.End direction="x" align="center">
          <Button.Button
            status="success"
            disabled={!allowEmpty && name.length === 0}
            onClick={() => {
              if (allowEmpty && name.length === 0) return onFinish(null);
              if (!allowEmpty && name.length === 0) return setError("Name is required");
              return onFinish(name);
            }}
            triggers={SAVE_TRIGGER}
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </>
    );

    return (
      <ModalContentLayout footer={footer}>
        <Input.Item
          label="Name"
          required={!allowEmpty}
          helpText={error}
          helpTextVariant={error != null ? "error" : "success"}
          padHelpText
        >
          <Input.Text
            autoFocus
            placeholder="Name"
            level="h2"
            variant="natural"
            value={name}
            onChange={setName}
          />
        </Input.Item>
      </ModalContentLayout>
    );
  },
);
