// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/editor/CreateModal.css";

import { type arc } from "@synnaxlabs/client";
import { Button, CSS as PCSS, Icon, Input, Nav, Select, Text } from "@synnaxlabs/pluto";
import { useState } from "react";

import { CSS } from "@/css";
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

export const CREATE_ARC_LAYOUT_TYPE = "arc_create_modal";

const MODE_KEYS: arc.Mode[] = ["graph", "text"];

export interface ArcModeSelectButtonProps extends Select.ButtonProps<arc.Mode> {
  icon: Icon.ReactElement;
  title: string;
  description: string;
}

const ArcModeSelectButton = ({
  itemKey,
  icon,
  title,
  description,
  ...rest
}: ArcModeSelectButtonProps) => {
  const { selected, onSelect } = Select.useItemState<arc.Mode>(itemKey);
  return (
    <Button.Button
      y
      className={CSS(
        CSS.BE("arc-create-modal", "mode-select-button"),
        PCSS.selected(selected),
      )}
      contrast={2}
      onClick={onSelect}
      grow
      justify="start"
      {...rest}
    >
      <Text.Text>
        {icon} {title}
      </Text.Text>
      <Text.Text color={9} level="small" wrap overflow="wrap">
        {description}
      </Text.Text>
    </Button.Button>
  );
};

export const [useCreateModal, CreateModal] = createBase<CreateArcResult, CreateArcArgs>(
  "Arc.Create Automation",
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
      <ModalContentLayout footer={footer} className={CSS.B("arc-create-modal")}>
        <Input.Item
          label="Name"
          required
          helpText={error}
          status={error != null ? "error" : "success"}
          padHelpText
        >
          <Input.Text
            autoFocus
            placeholder="Automation Name"
            level="h2"
            variant="text"
            value={name}
            onChange={setName}
            selectOnFocus
          />
        </Input.Item>
        <Input.Item label="Editor Mode" padHelpText full="x">
          <Select.Buttons
            value={mode}
            onChange={setMode}
            keys={MODE_KEYS}
            pack={false}
            x
            full="x"
          >
            <ArcModeSelectButton
              itemKey="graph"
              icon={<Icon.Schematic />}
              title="Graph"
              description="Visual, block-based editor that is best for simple automations such as alarms"
            />
            <ArcModeSelectButton
              itemKey="text"
              icon={<Icon.Text />}
              title="Text"
              description="Text-based editor that is best for complex automations such as control sequences"
            />
          </Select.Buttons>
        </Input.Item>
      </ModalContentLayout>
    );
  },
  { window: { size: { height: 350, width: 650 }, navTop: true }, icon: "Arc" },
);
