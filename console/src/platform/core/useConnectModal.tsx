// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Form, Icon, type Input, Nav } from "@synnaxlabs/pluto";
import { type z } from "zod";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";
import { Session } from "@/session";

export interface ConnectModalParams {
  /** The Core being edited. Absent when adding a new one. */
  coreKey?: string;
}

const FORM_SCHEMA = Session.Core.coreZ.pick({
  name: true,
  host: true,
  port: true,
  secure: true,
});

const ZERO_VALUES: z.infer<typeof FORM_SCHEMA> = {
  name: "",
  host: "",
  port: 9090,
  secure: false,
};

const HOST_FIELD_PROPS: Partial<Input.TextProps> = {
  placeholder: "localhost",
};

export const useConnectModal = Modals.create<ConnectModalParams>(
  ({ coreKey, close }) => {
    const dispatch = Session.useDispatch();
    const isEdit = coreKey != null;
    const existing = Session.Core.useSelectState(coreKey);
    const methods = Form.use<typeof FORM_SCHEMA>({
      schema: FORM_SCHEMA,
      values:
        isEdit && existing != null
          ? {
              name: existing.name,
              host: existing.host,
              port: existing.port,
              secure: existing.secure,
            }
          : { ...ZERO_VALUES },
    });

    // A record is an address-book entry: reachability is proven later at login, so a
    // save must not wait on (or be gated by) a live connectivity check.
    const handleSubmit = (): void => {
      if (!methods.validate()) return;
      const data = methods.value();
      if (isEdit && existing != null)
        dispatch(
          Session.Core.set({
            ...data,
            key: coreKey,
            username: existing.username,
            password: existing.password,
          }),
        );
      else dispatch(Session.Core.set({ ...data, username: "", password: "" }));
      close();
    };

    return (
      <Modals.Frame className={CSS.B("connect-core")}>
        <Modals.Header icon={<Icon.Core />}>Connect a Core</Modals.Header>
        <Form.Form<typeof FORM_SCHEMA> {...methods}>
          <Modals.Body gap="tiny" align="stretch">
            <Form.TextField
              path="name"
              inputProps={{
                autoFocus: true,
                variant: "text",
                level: "h2",
                placeholder: "Synnax Core",
                grow: true,
              }}
            />
            <Flex.Box x align="stretch">
              <Form.TextField path="host" grow inputProps={HOST_FIELD_PROPS} />
              <Form.NumericField path="port" />
              <Form.SwitchField path="secure" />
            </Flex.Box>
          </Modals.Body>
        </Form.Form>
        <Modals.Footer>
          <Nav.Bar.Start gap="small">
            <Triggers.SaveHelpText action={isEdit ? "Save" : "Connect"} noBar />
          </Nav.Bar.Start>
          <Nav.Bar.End>
            <Button.Button
              onClick={handleSubmit}
              trigger={Triggers.SAVE}
              variant="filled"
            >
              {isEdit ? "Save" : "Connect"}
            </Button.Button>
          </Nav.Bar.End>
        </Modals.Footer>
      </Modals.Frame>
    );
  },
);
