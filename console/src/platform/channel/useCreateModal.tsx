// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, Telem } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useState } from "react";
import { type z } from "zod";

import { Analytics } from "@/platform/analytics";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const path = deep.path<z.infer<typeof Channel.formSchema>>();

const INDEX_QUERY: Partial<Channel.RetrieveMultipleQuery> = { isIndex: true };

type Kind = "data" | "index" | "virtual";

const Create = ({ close }: Modals.ContentProps<Record<never, never>, void>) => {
  const [createMore, setCreateMore] = useState(false);
  const { capture } = Analytics.use();
  const { form, variant, save } = Channel.useForm({
    query: null,
    afterSave: ({ reset }) => {
      capture("channel_created", { calculated: false });
      if (createMore) reset(Channel.ZERO_FORM_VALUES);
      else close();
    },
  });
  const isIndex = Form.useFieldValue(path.isIndex, { ctx: form });
  const isVirtual = Form.useFieldValue(path.virtual, { ctx: form });
  const kind: Kind = isIndex ? "index" : isVirtual ? "virtual" : "data";
  const handleKindChange = (next: Kind) => {
    form.set(path.isIndex, next === "index");
    form.set(path.virtual, next === "virtual");
    if (next !== "data") form.set(path.index, 0);
    if (next === "index") form.set(path.dataType, DataType.TIMESTAMP.toString());
    else if (isIndex) form.set(path.dataType, Channel.ZERO_FORM_VALUES.dataType);
  };
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Channel />}>Channel.Create</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof Channel.formSchema> {...form}>
          <Form.Field path={path.name} showLabel={false} padHelpText={false}>
            {(p) => (
              <Input.Text
                autoFocus
                level="h2"
                variant="text"
                placeholder="Channel name"
                aria-label="Channel name"
                {...p}
              />
            )}
          </Form.Field>
          <Flex.Box x gap="large">
            <Input.Item label="Kind">
              <Select.Buttons<Kind> value={kind} onChange={handleKindChange}>
                <Select.Item itemKey="data">
                  <Icon.Channel />
                  Data
                </Select.Item>
                <Select.Item itemKey="index">
                  <Icon.Time />
                  Index
                </Select.Item>
                <Select.Item itemKey="virtual">
                  <Icon.Virtual />
                  Virtual
                </Select.Item>
              </Select.Buttons>
            </Input.Item>
            <Form.Field path={path.dataType} label="Data type" grow>
              {(p) => (
                <Telem.SelectDataType {...p} disabled={isIndex} zIndex={100} full="x" />
              )}
            </Form.Field>
          </Flex.Box>
          <Form.Field path={path.index} label="Index" visible={kind === "data"}>
            {({ value, onChange }) => (
              <Channel.SelectSingle
                value={value}
                onChange={onChange}
                initialQuery={INDEX_QUERY}
                allowNone={false}
                zIndex={100}
              />
            )}
          </Form.Field>
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End align="center" gap="large">
          <Flex.Box x align="center" gap="small">
            <Input.Switch value={createMore} onChange={setCreateMore} />
            <Text.Text color={9}>Create more</Text.Text>
          </Flex.Box>
          <Button.Button
            status={variant}
            variant="filled"
            onClick={() => save()}
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

export const useCreateModal = Modals.create(Create);
