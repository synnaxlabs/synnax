// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Channel,
  Flex,
  Form,
  Input,
  Nav,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { type CalculatedLayoutArgs } from "@/channel/calculatedLayout";
import { Code } from "@/code";
import { Arc } from "@/code/arc";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const Calculated: Layout.Renderer = ({ layoutKey, onClose }): ReactElement => {
  const args = Layout.useSelectArgs<CalculatedLayoutArgs>(layoutKey);
  const isEdit = args?.channelKey !== 0;

  const { form, variant, save, status } = Channel.useCalculatedForm({
    query: { key: args?.channelKey },
    afterSave: ({ reset }) => {
      if (createMore) reset();
      else onClose();
    },
  });

  const [createMore, setCreateMore] = useState(false);

  if (variant !== "success") return <Status.Summary status={status} />;

  return (
    <Flex.Box className={CSS.B("channel-edit-layout")} grow empty>
      <Flex.Box className="console-form" style={{ padding: "3rem" }} grow>
        <Form.Form<typeof Channel.calculatedFormSchema> {...form}>
          <Form.Field<string> path="name" label="Name">
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

          <Form.Field<string> path="expression" grow>
            {({ value, onChange }) => (
              <Code.Editor
                value={value}
                language={Arc.LANGUAGE}
                onChange={onChange}
                isBlock
                bordered
                rounded
                style={{ height: 150 }}
              />
            )}
          </Form.Field>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action={isEdit ? "Save" : "Create"} />
        <Nav.Bar.End align="center" gap="large">
          {isEdit && (
            <Flex.Box x align="center" gap="small">
              <Input.Switch value={createMore} onChange={setCreateMore} />
              <Text.Text color={9}>Create More</Text.Text>
            </Flex.Box>
          )}
          <Flex.Box x align="center">
            <Button.Button
              status={variant}
              trigger={Triggers.SAVE}
              variant="filled"
              onClick={() => save()}
            >
              {isEdit ? "Save" : "Create"}
            </Button.Button>
          </Flex.Box>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
