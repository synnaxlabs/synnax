// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/Connect.css";

import { checkConnection, type connection, Synnax as Client } from "@synnaxlabs/client";
import { Button, Flex, Form, Input, Nav, Status, Synnax } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { useState } from "react";
import { useDispatch } from "react-redux";
import { type z } from "zod";

import { useSelectAllNames } from "@/cluster/selectors";
import { clusterZ, set, setActive } from "@/cluster/slice";
import { testConnection } from "@/cluster/testConnection";
import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";
import { useCreateOrRetrieve } from "@/workspace/useCreateOrRetrieve";

export const CONNECT_LAYOUT_TYPE = "connectCluster";

export const CONNECT_LAYOUT: Layout.BaseState = {
  key: CONNECT_LAYOUT_TYPE,
  type: CONNECT_LAYOUT_TYPE,
  name: "Cluster.Connect",
  icon: "Cluster",
  location: "modal",
  window: { resizable: false, size: { height: 300, width: 650 }, navTop: true },
};

const baseFormSchema = clusterZ.pick({
  name: true,
  host: true,
  port: true,
  secure: true,
});

const ZERO_VALUES: z.infer<typeof baseFormSchema> = {
  name: "",
  host: "",
  port: "",
  secure: false,
};

/**
 * Connect implements the LayoutRenderer component type to provide a form for connecting
 * to a cluster.
 */
export const Connect: Layout.Renderer = ({ onClose }) => {
  const dispatch = useDispatch();
  const [connState, setConnState] = useState<connection.State | null>(null);
  const [loading, setLoading] = useState<"test" | "submit" | null>(null);
  const names = useSelectAllNames();
  const formSchema = baseFormSchema.check(({ value: { name }, issues }) => {
    if (names.includes(name))
      issues.push({
        input: name,
        code: "custom",
        path: ["name"],
        message: `${name} is already in use.`,
      });
  });
  const handleError = Status.useErrorHandler();
  const methods = Form.use<typeof formSchema>({
    schema: formSchema,
    values: { ...ZERO_VALUES },
  });

  const handleSubmit = (): void =>
    handleError(async () => {
      if (!methods.validate()) return;
      const data = methods.value();
      setConnState(null);
      setLoading("submit");
      await checkConnection(data);
      setLoading(null);
      setConnState({
        status: "connected",
        message: "Connected to cluster",
        clusterKey: "123",
        nodeVersion: "1.0.0",
        clientVersion: "1.0.0",
        clientServerCompatible: true,
      });
      setTimeout(() => {
        const clusterProps = { ...data, key: "123" };
        dispatch(set({ ...clusterProps, username: "", password: "" }));
        onClose();
      }, 500);
    }, "Failed to connect to cluster");

  return (
    <Flex.Box grow className={CSS.B("connect-cluster")}>
      <Form.Form<typeof formSchema> {...methods}>
        <Flex.Box
          className="console-form"
          grow
          gap="tiny"
          justify="center"
          align="stretch"
        >
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
            <Form.Field<string> path="host" grow>
              {(p) => <Input.Text placeholder="localhost" {...p} full="x" />}
            </Form.Field>
            <Form.Field<string> path="port">
              {(p) => <Input.Text placeholder="9090" {...p} />}
            </Form.Field>
          </Flex.Box>
        </Flex.Box>
      </Form.Form>
      <Modals.BottomNavBar>
        <Nav.Bar.Start gap="small">
          {connState != null ? (
            <Status.Summary
              variant={Synnax.CONNECTION_STATE_VARIANTS[connState.status]}
            >
              {connState.status === "connected"
                ? caseconv.capitalize(connState.status)
                : connState.message}
            </Status.Summary>
          ) : (
            <Triggers.SaveHelpText action="Connect" noBar />
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            onClick={handleSubmit}
            status={loading === "submit" ? "loading" : undefined}
            trigger={Triggers.SAVE}
            variant="filled"
          >
            Connect
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
