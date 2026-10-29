// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/range/Create.css";

import { TimeStamp } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Ranger, Synnax } from "@synnaxlabs/pluto";
import { type NumericTimeRange, TimeRange, uuid } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { type z } from "zod";

import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Label } from "@/platform/label";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";
import { Session } from "@/session";

export type CreateModalParams = Omit<
  Partial<z.infer<typeof Ranger.createFormSchema>>,
  "key"
>;

const ParentRangeIcon = Icon.createComposite(Icon.Range, {
  bottomRight: Icon.Arrow.Up,
});

// A parent is a range, but "Range" alone would not say which one, so this trigger
// names itself. It reads as content, like the timeline above it.
const PARENT_TRIGGER_PROPS = { placeholder: "Parent range", variant: "text" as const };

const TimelineField = (): ReactElement => {
  const parentKey = Form.useFieldValue<string>("parent");
  const parent = Ranger.useResult(parentKey === "" ? null : { key: parentKey });
  const parentRange = parent.data?.timeRange.numeric;
  return (
    <Form.Field<NumericTimeRange>
      path="timeRange"
      showLabel={false}
      padHelpText={false}
    >
      {(p) => (
        <Ranger.Timeline level="h4" variant="shadow" parent={parentRange} {...p} />
      )}
    </Form.Field>
  );
};

const Create = ({ close, ...params }: Modals.ContentProps<CreateModalParams, void>) => {
  const [now] = useState(() => Number(TimeStamp.now().valueOf()));
  const dispatch = Session.useDispatch();
  const { capture } = Analytics.use();

  const client = Synnax.use();
  const clientExists = client != null;
  const { form, save, variant } = Ranger.useCreateForm({
    query: null,
    autoSave: false,
    initialValues: {
      key: uuid.create(),
      name: "",
      labels: [],
      timeRange: { start: now, end: TimeStamp.MAX.nanoseconds },
      parent: "",
      ...params,
    },
    afterSave: (form) => {
      capture("range_created", {});
      close();
      const { key } = form.value();
      if (key == null) return;
      dispatch(Session.Range.add({ variant: "persisted", key }));
    },
  });

  const saveLocal = () => {
    if (!form.validate()) return;
    const { name, key, timeRange } = form.value();
    if (key == null) return;
    dispatch(
      Session.Range.add({
        variant: "static",
        key,
        name,
        timeRange: new TimeRange(timeRange.start, timeRange.end).numeric,
      }),
    );
    close();
  };

  const saveName = "Save to Core";

  return (
    <Modal.Frame className={CSS.B("range-create-layout")}>
      <Modal.Header icon={<Icon.Range />}>Range.Create</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof Ranger.createFormSchema> {...form}>
          <Flex.Box y gap="huge">
            <Form.Field<string> path="name" showLabel={false} padHelpText={false}>
              {(p) => (
                <Input.Text
                  autoFocus
                  level="h2"
                  variant="text"
                  placeholder="Range name"
                  aria-label="Range name"
                  {...p}
                />
              )}
            </Form.Field>
            <TimelineField />
            <Flex.Box x align="center">
              <Form.Field<string>
                path="parent"
                visible
                showLabel={false}
                padHelpText={false}
              >
                {({ onChange, value }) => (
                  <Ranger.Select
                    className={CSS.BE("range-create-layout", "parent")}
                    value={value}
                    // Clearing the selection reports null, but the schema types an
                    // absent parent as "". Null fails validation on a field that
                    // hides itself when null, so the save would stall unexplained.
                    onChange={(v: string | null) => onChange(v ?? "")}
                    icon={<ParentRangeIcon />}
                    triggerProps={PARENT_TRIGGER_PROPS}
                    allowNone
                  />
                )}
              </Form.Field>
              <Form.Field<string[]> path="labels" showLabel={false} padHelpText={false}>
                {(p) => <Label.SelectMultiple zIndex={100} {...p} />}
              </Form.Field>
            </Flex.Box>
          </Flex.Box>
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
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
            triggerIndicator
          >
            {saveName}
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export const useCreateModal = Modals.create<CreateModalParams>(Create);
