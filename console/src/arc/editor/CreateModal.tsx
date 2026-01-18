// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { Button, Input, Nav, Select } from "@synnaxlabs/pluto";
import { useState } from "react";

import { type BaseArgs, createBase } from "@/modals/Base";
import { ModalContentLayout } from "@/modals/layout";
import { Triggers } from "@/triggers";

export interface CreateArcResult {
  name: string;
  mode: arc.Mode;
}

export interface CreateArcArgs extends BaseArgs<CreateArcResult> {
  initialName?: string;
  initialMode?: arc.Mode;
}

export const CREATE_ARC_LAYOUT_TYPE = "arc_create";

const MODE_KEYS: arc.Mode[] = ["graph", "text"];

export const [useCreate, Create] = createBase<CreateArcResult, CreateArcArgs>(
  "Create Arc Program",
  CREATE_ARC_LAYOUT_TYPE,
  ({ value: { result, initialName, initialMode }, onFinish }) => {
    const [name, setName] = useState(result?.name ?? initialName ?? "");
    const [mode, setMode] = useState<arc.Mode>(result?.mode ?? initialMode ?? "graph");
    const [error, setError] = useState<string | undefined>(undefined);

    const handleSave = () => {
      if (name.length === 0) return setError("Name is required");
      onFinish({ name, mode });
    };

    const footer = (
      <>
        <Triggers.SaveHelpText action="Create" trigger={Triggers.SAVE} />
        <Nav.Bar.End align="center">
          <Button.Button
            status="success"
            disabled={name.length === 0}
            variant="filled"
            onClick={handleSave}
            trigger={Triggers.SAVE}
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
          required
          helpText={error}
          status={error != null ? "error" : "success"}
          padHelpText
        >
          <Input.Text
            autoFocus
            placeholder="Arc Program Name"
            level="h2"
            variant="text"
            value={name}
            onChange={setName}
            selectOnFocus
          />
        </Input.Item>
        <Input.Item label="Editor Mode" padHelpText>
          <Select.Buttons value={mode} onChange={setMode} keys={MODE_KEYS}>
            <Select.Button itemKey="graph">Graph</Select.Button>
            <Select.Button itemKey="text">Text</Select.Button>
          </Select.Buttons>
        </Input.Item>
      </ModalContentLayout>
    );
  },
  { window: { size: { height: 300, width: 500 } } },
);
