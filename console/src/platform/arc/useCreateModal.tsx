// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/arc/CreateModal.css";

import { type arc, status, UnexpectedError } from "@synnaxlabs/client";
import {
  Arc,
  Button,
  CSS as PCSS,
  Form,
  Icon,
  Input,
  Nav,
  Select,
  Text,
} from "@synnaxlabs/pluto";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

export interface CreateModalResult {
  key: arc.Key;
}

export interface CreateModalParams {
  initialName?: string;
  initialMode?: arc.Mode;
}

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

export const useCreateModal = Modals.createPrompt<CreateModalResult, CreateModalParams>(
  ({ initialName, initialMode, close }) => {
    const { form, save, variant } = Arc.useForm({
      query: {},
      initialValues: { name: initialName ?? "", mode: initialMode ?? "graph" },
      afterSave: ({ value }) => {
        const { key } = value();
        if (key == null) throw new UnexpectedError("Arc key is null");
        close({ key });
      },
    });

    return (
      <Modals.Frame className={CSS.B("arc-create-modal")}>
        <Modals.Header icon={<Icon.Arc />}>Arc.Create Automation</Modals.Header>
        <Modals.Body>
          <Form.Form<typeof Arc.formSchema> {...form}>
            <Form.Field<string> path="name" required>
              {(p) => (
                <Input.Text
                  autoFocus
                  placeholder="Automation Name"
                  level="h2"
                  variant="text"
                  selectOnFocus
                  {...p}
                />
              )}
            </Form.Field>
            <Form.Field<arc.Mode> path="mode" label="Editor Mode" full="x">
              {({ value, onChange }) => (
                <Select.Buttons
                  value={value}
                  onChange={onChange}
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
              )}
            </Form.Field>
          </Form.Form>
        </Modals.Body>
        <Modals.Footer>
          <Triggers.SaveHelpText action="Create" trigger={Triggers.SAVE} />
          <Nav.Bar.End align="center">
            <Button.Button
              status={status.keepVariants(variant, "loading")}
              variant="filled"
              onClick={() => save()}
              trigger={Triggers.SAVE}
            >
              Create
            </Button.Button>
          </Nav.Bar.End>
        </Modals.Footer>
      </Modals.Frame>
    );
  },
);
