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
import { Button } from "@synnaxlabs/lyra/button";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Input } from "@synnaxlabs/lyra/input";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Arc, type Flux } from "@synnaxlabs/pluto";
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

const NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  autoFocus: true,
  placeholder: "Name",
  level: "h2",
  variant: "text",
  selectOnFocus: true,
};

const MODE_CLASS = CSS.BE("arc-create-modal", "mode-select-button");

export const useCreateModal = Modals.createPrompt<CreateModalResult, CreateModalParams>(
  ({ initialValues, close }) => {
    const { form, save, variant } = Arc.useForm({
      query: null,
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
        <Modals.Header icon={<Icon.Arc />}>Arc.Create</Modals.Header>
        <Modals.Body>
          <Form.Form<typeof Arc.formSchema> {...form}>
            <Form.TextField path="name" required inputProps={NAME_INPUT_PROPS} />
            <Form.Field<arc.Mode> path="mode" label="Editor mode" full="x">
              {({ value, onChange }) => (
                <Select.Buttons
                  value={value}
                  onChange={onChange}
                  pack={false}
                  x
                  full="x"
                >
                  <Select.Item
                    itemKey="text"
                    className={MODE_CLASS}
                    variant="outlined"
                    y
                    grow
                    alignSelf="stretch"
                    justify="start"
                  >
                    <Text.Text>
                      <Icon.Text /> Text
                    </Text.Text>
                    <Text.Text color={9} level="small" wrap overflow="wrap">
                      Best for complex automations such as control sequences
                    </Text.Text>
                  </Select.Item>
                  <Select.Item
                    itemKey="graph"
                    className={MODE_CLASS}
                    variant="outlined"
                    y
                    grow
                    alignSelf="stretch"
                    justify="start"
                  >
                    <Text.Text>
                      <Icon.Schematic /> Graph
                    </Text.Text>
                    <Text.Text color={9} level="small" wrap overflow="wrap">
                      Best for simple automations such as alarms
                    </Text.Text>
                  </Select.Item>
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
