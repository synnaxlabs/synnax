// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/range/Create.css";

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
} from "@synnaxlabs/pluto";
import { type NumericTimeRange, TimeRange, uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef } from "react";
import { type z } from "zod";

import { CSS } from "@/platform/css";
import { Label } from "@/platform/label";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";
import { Session } from "@/session";

export type CreateModalParams = Omit<
  Partial<z.infer<typeof Ranger.formSchema>>,
  "key"
> & {
  /** When provided, the modal edits the existing range with this key. */
  rangeKey?: string;
};

const ParentRangeIcon = Icon.createComposite(Icon.Range, {
  bottomRight: Icon.Arrow.Up,
});

const TimelineField = (): ReactElement => {
  const parentKey = Form.useFieldValue<string>("parent");
  const parent = Ranger.useResult(parentKey === "" ? null : { key: parentKey });
  const parentRange = parent.data?.timeRange.numeric;
  return (
    <Form.Field<NumericTimeRange> path="timeRange" label="Stage" padHelpText={false}>
      {(p) => <Ranger.Timeline level="h4" parent={parentRange} {...p} />}
    </Form.Field>
  );
};

export const useCreateModal = Modals.create<CreateModalParams>(
  ({ close, rangeKey, ...params }) => {
    const now = useRef(Number(TimeStamp.now().valueOf())).current;
    const dispatch = Session.useDispatch();

    const client = Synnax.use();
    const clientExists = client != null;
    const { form, save, variant } = Ranger.useForm({
      query: rangeKey == null ? null : { key: rangeKey },
      autoSave: false,
      initialValues: {
        key: rangeKey ?? uuid.create(),
        name: "",
        labels: [],
        timeRange: { start: now, end: TimeStamp.MAX.nanoseconds },
        parent: "",
        ...params,
      },
      afterSave: (form) => {
        close();
        const { name, key, timeRange } = form.value();
        if (key == null) return;
        dispatch(
          Session.Range.add({
            name,
            key,
            persisted: true,
            variant: "static",
            timeRange,
          }),
        );
      },
    });

    const saveLocal = useCallback(() => {
      if (!form.validate()) return;
      const { name, key, timeRange } = form.value();
      if (key == null) return;
      dispatch(
        Session.Range.add({
          persisted: false,
          name,
          key,
          variant: "static",
          timeRange: new TimeRange(timeRange.start, timeRange.end).numeric,
        }),
      );
      close();
    }, [form, dispatch]);

    // Makes sure the user doesn't have the option to select the range itself as a parent
    const recursiveParentFilter = useCallback(
      (data: ranger.Payload) => data.key !== rangeKey,
      [rangeKey],
    );

    const saveName = "Save to Core";

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
                  placeholder="Name"
                  {...p}
                />
              )}
            </Form.Field>
            <TimelineField />
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
                {(p) => <Label.SelectMultiple zIndex={100} {...p} />}
              </Form.Field>
            </Flex.Box>
          </Form.Form>
        </Modals.Body>
        <Modals.Footer>
          <Triggers.SaveHelpText action={saveName} />
          <Nav.Bar.End>
            <Button.Button onClick={() => saveLocal()} disabled={variant === "loading"}>
              Save locally
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
