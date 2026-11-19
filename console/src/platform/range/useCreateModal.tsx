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

const Create = ({
  close,
  ...params
}: Modals.ContentProps<CreateModalParams, ranger.Key>) => {
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
    afterSave: (form) => {
      capture("range_created", {});
      close(form.value().key);
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
                    icon={<Ranger.ParentIcon />}
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

/** Opens a dialog that creates a range, and resolves with its key. */
export const useCreateModal = Modals.createPrompt<ranger.Key, CreateModalParams>(
  Create,
);
