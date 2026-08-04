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
  type Flux,
  Form,
  Icon,
  type Input,
  Nav,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";
import { type z } from "zod";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

export interface CreateModalResult {
  key: arc.Key;
}

export interface CreateModalParams {
  initialValues?: Partial<z.infer<typeof Arc.formSchema>>;
}

const MODE_KEYS: arc.Mode[] = ["text", "graph"];

const QUERY: Arc.FormQuery = {};

const NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  autoFocus: true,
  placeholder: "Automation Name",
  level: "h2",
  variant: "text",
  selectOnFocus: true,
};

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
  ({ initialValues, close }) => {
    const { form, save, variant } = Arc.useForm({
      query: QUERY,
      initialValues: useMemo(
        () => ({ ...Arc.ZERO_FORM_VALUES, ...initialValues }),
        [initialValues],
      ),
      afterSave: useCallback(
        ({ value }: Flux.AfterSaveParams<Arc.FormQuery, typeof Arc.formSchema>) => {
          const { key } = value();
          if (key == null) throw new UnexpectedError("Arc key is null");
          close({ key });
        },
        [close],
      ),
    });

    return (
      <Modals.Frame className={CSS.B("arc-create-modal")}>
        <Modals.Header icon={<Icon.Arc />}>Arc.Create Automation</Modals.Header>
        <Modals.Body>
          <Form.Form<typeof Arc.formSchema> {...form}>
            <Form.TextField path="name" required inputProps={NAME_INPUT_PROPS} />
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
                    itemKey="text"
                    icon={<Icon.Text />}
                    title="Text"
                    description="Text-based editor that is best for complex automations such as control sequences"
                  />
                  <ArcModeSelectButton
                    itemKey="graph"
                    icon={<Icon.Schematic />}
                    title="Graph"
                    description="Visual, block-based editor that is best for simple automations such as alarms"
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
