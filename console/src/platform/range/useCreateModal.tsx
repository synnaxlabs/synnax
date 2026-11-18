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
import { primitive, uuid } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { type z } from "zod";

import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Label } from "@/platform/label";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const path = Ranger.createFormPath;

export type CreateModalParams = Omit<
  Partial<z.infer<typeof Ranger.createFormSchema>>,
  "key"
>;

const ParentRangeIcon = Icon.createFromPath(
  "range-parent",
  "M16.4,13.23c0,0.04 0,0.08 -0.01,0.12c-0.01,0.16 -0.03,0.32 -0.05,0.47c0,0 0,0 0,0c-0.17,1.12 -0.7,2.17 -1.51,2.99c-2.12,2.11 -5.53,2.11 -7.64,0l3.82,-3.82v-5.38c0.02,0 0.05,-0.01 0.07,-0.01c0.03,0 0.07,0.01 0.1,0.02c0.45,2.55 2.39,4.58 4.9,5.14c0.19,0.07 0.33,0.26 0.33,0.47zM11.37,4.59c-0.09,0.27 -0.15,0.55 -0.2,0.84c-0.05,0.2 -0.23,0.35 -0.44,0.37c-0.02,0 -0.04,0 -0.07,0c-0.18,0.01 -0.37,0.02 -0.55,0.04c0,0 0,0 0,0c-3.56,0.44 -6.31,3.47 -6.31,7.15c0,3.98 3.22,7.2 7.2,7.2c3.67,0 6.7,-2.75 7.14,-6.3c0,0 0,0 0,0c0.02,-0.18 0.04,-0.36 0.04,-0.54c0,-0.02 0,-0.05 0,-0.07c0.02,-0.21 0.17,-0.39 0.37,-0.44c0.29,-0.05 0.57,-0.11 0.84,-0.2c0.14,-0.03 0.28,0.01 0.4,0.09c0.13,0.09 0.2,0.24 0.2,0.4c0,0 0,0.01 0,0.01c0,0.11 -0.01,0.22 -0.01,0.33c0,0 0,0 0,0c-0.25,4.74 -4.18,8.52 -8.99,8.52c-4.97,0 -9,-4.03 -9,-9c0,-4.81 3.78,-8.74 8.53,-8.99c0,0 0,0 0,0c0.11,-0.01 0.22,-0.01 0.33,-0.01c0,0 0.01,0 0.01,0c0.16,0 0.3,0.07 0.4,0.2c0.09,0.11 0.12,0.26 0.09,0.4zM17.5,2l4.5,4.95h-3.25v4.05h-2.5v-4.05h-3.25z",
);

// A parent is a range, but "Range" alone would not say which one, so this trigger
// names itself. It reads as content, like the timeline above it.
const PARENT_TRIGGER_PROPS = { placeholder: "Parent range", variant: "text" as const };

const TimelineField = (): ReactElement => {
  const parentKey = Form.useFieldValue(path.parent);
  const parent = Ranger.useResult(
    primitive.isNonZero(parentKey) ? { key: parentKey } : null,
  );
  const parentRange = parent.data?.timeRange.numeric;
  return (
    <Form.Field path={path.timeRange} showLabel={false} padHelpText={false}>
      {(p) => (
        <Ranger.Timeline level="h4" variant="shadow" parent={parentRange} {...p} />
      )}
    </Form.Field>
  );
};

const Create = ({ close, ...params }: Modals.ContentProps<CreateModalParams, void>) => {
  const [now] = useState(() => Number(TimeStamp.now().valueOf()));
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
    afterSave: () => {
      capture("range_created", {});
      close();
    },
  });
  Modals.useBusy(variant === "loading");

  return (
    <Modal.Frame className={CSS.B("range-create-layout")}>
      <Modal.Header icon={<Icon.Range />}>Range.Create</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof Ranger.createFormSchema> {...form}>
          <Flex.Box y gap="huge">
            <Form.Field path={path.name} showLabel={false} padHelpText={false}>
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
              <Form.Field
                path={path.parent}
                visible
                showLabel={false}
                showHelpText={false}
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
              <Form.Field path={path.labels} showLabel={false} showHelpText={false}>
                {(p) => <Label.SelectMultiple {...p} />}
              </Form.Field>
            </Flex.Box>
          </Flex.Box>
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            variant="filled"
            onClick={() => save()}
            disabled={!clientExists}
            status={variant}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export const useCreateModal = Modals.create<CreateModalParams>(Create);
