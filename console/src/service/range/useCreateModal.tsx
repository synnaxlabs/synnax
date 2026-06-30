// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Create.css";

import { type ranger, TimeStamp } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Form,
  Icon,
  Input,
  Nav,
  Ranger,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type NumericTimeRange, TimeRange, uuid } from "@synnaxlabs/x";
import { useCallback, useRef } from "react";
import { useDispatch } from "react-redux";
import { type z } from "zod";

import { CSS } from "@/component/css";
import { Label } from "@/service/label";
import { Modals } from "@/component/modals";
import { add } from "@/session/range/slice";
import { Triggers } from "@/component/triggers";

export type CreateModalParams = Partial<z.infer<typeof Ranger.formSchema>>;

const ParentRangeIcon = Icon.createComposite(Icon.Range, {
  bottomRight: Icon.Arrow.Up,
});

export const useCreateModal = Modals.create<CreateModalParams>(
  ({ close, ...params }) => {
    const now = useRef(Number(TimeStamp.now().valueOf())).current;
    const dispatch = useDispatch();

    const client = Synnax.use();
    const clientExists = client != null;
    const { form, save, variant } = Ranger.useForm({
      query: { key: params.key },
      autoSave: false,
      initialValues: {
        key: uuid.create(),
        name: "",
        labels: [],
        timeRange: { start: now, end: now },
        parent: "",
        ...params,
      },
      afterSave: (form) => {
        close();
        const { name, key, timeRange } = form.value();
        if (key == null) return;
        dispatch(
          add({
            ranges: [{ name, key, persisted: true, variant: "static", timeRange }],
          }),
        );
      },
    });

    const saveLocal = useCallback(() => {
      if (!form.validate()) return;
      const value = form.value();
      if (value.key == null) return;
      dispatch(
        add({
          ranges: [
            {
              persisted: false,
              ...value,
              key: value.key ?? "",
              variant: "static",
              timeRange: new TimeRange(value.timeRange.start, value.timeRange.end)
                .numeric,
            },
          ],
        }),
      );
      close();
    }, [form, dispatch]);

    // Makes sure the user doesn't have the option to select the range itself as a parent
    const recursiveParentFilter = useCallback(
      (data: ranger.Payload) => data.key !== params.key,
      [params.key],
    );

    const saveName = "Save to Synnax";

    return (
      <Modals.Frame className={CSS.B("range-create-layout")}>
        <Modals.Header icon={<Icon.Range />}>Range.Create</Modals.Header>
        <Modals.Body>
          <Form.Form<typeof Ranger.formSchema> {...form}>
            <Form.Field<string> path="name">
              {(p) => (
                <Input.Text
                  autoFocus
                  level="h2"
                  variant="text"
                  placeholder="Range Name"
                  {...p}
                />
              )}
            </Form.Field>
            <Form.Field<NumericTimeRange> path="timeRange" label="Stage">
              {(p) => (
                <Ranger.SelectStage
                  {...Ranger.wrapNumericTimeRangeToStage(p)}
                  className={CSS.BE("range-create-layout", "stage")}
                  triggerProps={{ variant: "outlined" }}
                />
              )}
            </Form.Field>
            <Flex.Box
              x
              wrap
              gap="large"
              className={CSS.BE("range-create-layout", "time-range")}
            >
              <Form.Field<number> path="timeRange.start" label="From">
                {(p) => <Input.DateTime level="h4" variant="text" {...p} />}
              </Form.Field>
              <Text.Text
                level="h4"
                className={CSS.BE("range-create-layout", "time-range-arrow")}
              >
                <Icon.Arrow.Right />
              </Text.Text>
              <Form.Field<number> path="timeRange.end" label="To">
                {(p) => <Input.DateTime level="h4" variant="text" {...p} />}
              </Form.Field>
            </Flex.Box>
            <Flex.Box x>
              <Form.Field<string> path="parent" visible padHelpText={false}>
                {({ onChange, value }) => (
                  <Ranger.Select
                    className={CSS.BE("range-create-layout", "parent")}
                    zIndex={-1}
                    filter={recursiveParentFilter}
                    value={value}
                    onChange={onChange}
                    icon={<ParentRangeIcon />}
                    allowNone
                  />
                )}
              </Form.Field>
              <Form.Field<string[]> path="labels" required={false}>
                {({ variant, ...p }) => <Label.SelectMultiple zIndex={100} {...p} />}
              </Form.Field>
            </Flex.Box>
          </Form.Form>
        </Modals.Body>
        <Modals.Footer>
          <Triggers.SaveHelpText action={saveName} />
          <Nav.Bar.End>
            <Button.Button onClick={() => saveLocal()} disabled={variant === "loading"}>
              Save Locally
            </Button.Button>
            <Button.Button
              variant="filled"
              onClick={() => save()}
              disabled={!clientExists}
              status={variant}
              trigger={Triggers.SAVE}
            >
              {saveName}
            </Button.Button>
          </Nav.Bar.End>
        </Modals.Footer>
      </Modals.Frame>
    );
  },
);
