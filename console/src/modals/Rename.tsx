// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Button, Input, Nav } from "@synnaxlabs/pluto";
import { useState } from "react";

import { Layout } from "@/layout";
import { type BaseArgs, createBase, type Prompt } from "@/modals/Base";
import { Triggers } from "@/triggers";

export interface PromptRenameLayoutArgs extends BaseArgs<string> {
  result?: string;
  allowEmpty?: boolean;
}

export const RENAME_LAYOUT_TYPE = "rename";

export interface PromptRename extends Prompt<string, PromptRenameLayoutArgs> {}

export const [useRename, Rename] = createBase<string, PromptRenameLayoutArgs>(
  "Name",
  RENAME_LAYOUT_TYPE,
  ({ value: { result, allowEmpty = false }, onFinish }) => {
    const [name, setName] = useState(result ?? "");
    const [error, setError] = useState<string | undefined>(undefined);
    return (
      <Align.Space direction="y" grow justify="center">
        <Align.Space
          direction="y"
          grow
          align="start"
          justify="center"
          style={{ padding: "5rem" }}
        >
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
        </Align.Space>
        <Layout.BottomNavBar>
          <Triggers.SaveHelpText action="Save" trigger={["Enter"]} />
          <Nav.Bar.End direction="x" align="center">
            <Button.Button
              status="success"
              disabled={!allowEmpty && name.length === 0}
              onClick={() => {
                if (allowEmpty && name.length === 0) return onFinish(null);
                if (!allowEmpty && name.length === 0)
                  return setError("Name is required");
                return onFinish(name);
              }}
              triggers={["Enter"]}
            >
              Create
            </Button.Button>
          </Nav.Bar.End>
        </Layout.BottomNavBar>
      </Align.Space>
    );
  },
);
