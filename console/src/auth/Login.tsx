// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax as Client } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import { Button, Flex, Form, Status, Text, type Triggers } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { Cluster } from "@/cluster";
import { setActive } from "@/cluster/slice";
import { Layout } from "@/layout";
import { Layouts } from "@/layouts";
import { Version } from "@/version";

const SIGN_IN_TRIGGER: Triggers.Trigger = ["Enter"];

const credentialsZ = z.object({
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

export interface Credentials extends z.infer<typeof credentialsZ> {}

export const Login = (): ReactElement => {
  const [stat, setStatus] = useState<status.Status>(() =>
    status.create({ variant: "disabled", message: "" }),
  );
  const initialSelected = Cluster.useSelectMany()[0]?.key;
  const [selectedKey, setSelectedKey] = useState<string | undefined>(initialSelected);
  const selectedCluster = Cluster.useSelect(selectedKey);
  const dispatch = useDispatch();
  const handleError = Status.useErrorHandler();

  const methods = Form.use<typeof credentialsZ>({
    schema: credentialsZ,
    values: { username: "", password: "" },
    onChange: () => {
      const usernameTouched = methods.get("username").touched;
      const passwordTouched = methods.get("password").touched;
      setStatus(
        status.create({
          variant: usernameTouched && passwordTouched ? "success" : "disabled",
          message: "",
        }),
      );
    },
  });

  const handleSubmit = (): void =>
    handleError(async () => {
      if (!methods.validate() || selectedCluster == null) return;
      const credentials = methods.value();
      setStatus(status.create({ variant: "loading", message: "Connecting..." }));
      const client = new Client({ ...selectedCluster, ...credentials });
      const state = await client.connectivity.check();
      if (state.status !== "connected") {
        const message = state.message ?? "Unknown error";
        return setStatus(status.create({ variant: "error", message }));
      }
      if (state.nodeVersion != null) dispatch(Version.set(state.nodeVersion));
      dispatch(Cluster.set({ ...selectedCluster, ...credentials }));
      dispatch(setActive(selectedCluster.key));
    }, "Failed to log in");

  return (
    <>
      <Layouts.Notifications />
      <Layout.Modals />
      <Flex.Box center background={1} gap="huge">
        <Logo variant="title" style={{ height: "10rem" }} />
        <Flex.Box
          pack
          x
          style={{ width: "800px", height: "400px" }}
          grow={false}
          rounded={1.5}
          background={0}
        >
          <Cluster.List value={selectedKey} onChange={setSelectedKey} />
          <Flex.Box
            y
            gap="huge"
            style={{ padding: "10rem" }}
            bordered
            grow
            shrink={false}
          >
            <Form.Form<typeof credentialsZ> {...methods}>
              <Flex.Box y align="center" grow gap="huge" shrink={false}>
                <Text.Text level="h2" color={11} weight={450}>
                  Log in to {selectedCluster?.name}
                </Text.Text>
                <Flex.Box y full="x" empty>
                  <Form.TextField
                    path="username"
                    inputProps={{
                      placeholder: "synnax",
                      autoFocus: true,
                      size: "large",
                    }}
                  />
                  <Form.TextField
                    path="password"
                    inputProps={{
                      placeholder: "seldon",
                      type: "password",
                      size: "large",
                    }}
                  />
                </Flex.Box>
                <Flex.Box style={{ height: "5rem" }}>
                  {stat.message !== "" && (
                    <Status.Summary variant={stat.variant} message={stat.message} />
                  )}
                </Flex.Box>
                <Button.Button
                  onClick={handleSubmit}
                  status={stat.variant}
                  trigger={SIGN_IN_TRIGGER}
                  variant="filled"
                  size="large"
                >
                  Log In
                </Button.Button>
              </Flex.Box>
            </Form.Form>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </>
  );
};
