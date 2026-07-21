// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/auth/Login.css";

import { Logo } from "@synnaxlabs/media";
import { Button, Flex, Form, Text } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import {
  credentialsZ,
  PASSWORD_INPUT_PROPS,
  SIGN_IN_TRIGGER,
  USERNAME_INPUT_PROPS,
} from "@/feature/auth/CredentialsForm";
import { LoginNav } from "@/feature/auth/LoginNav";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

export const Login = (): ReactElement => {
  const servingCluster = Cluster.detectConnection();
  const clusters = Session.Cluster.useSelectMany();
  const [selectedKey, setSelectedKey] = useState<string | undefined>(clusters[0]?.key);
  const selectedCluster = Session.Cluster.useSelectState(selectedKey);
  const dispatch = Session.useDispatch();

  const methods = Form.use<typeof credentialsZ>({
    schema: credentialsZ,
    values: { username: "", password: "" },
  });

  const handleSubmit = (): void => {
    const clusterToConnect = servingCluster ?? selectedCluster;
    if (!methods.validate() || clusterToConnect == null) return;
    const credentials = methods.value();
    const key =
      servingCluster == null && selectedCluster != null
        ? selectedCluster.key
        : uuid.create();
    dispatch(Session.Cluster.set({ ...clusterToConnect, key, ...credentials }));
    dispatch(Session.Cluster.select(key));
  };

  const handleSelectedClusterChange = useCallback(
    (key?: string) => {
      if (key == null) return;
      methods.reset();
      setSelectedKey(key);
    },
    [methods],
  );

  return (
    <Flex.Box y empty className={CSS.B("login")}>
      <LoginNav />
      <Flex.Box
        y
        align="center"
        justify="center"
        background={1}
        gap="huge"
        grow
        data-tauri-drag-region
        className={CSS.BE("login", "content")}
      >
        <Logo
          variant="title"
          className={CSS.BE("login", "logo")}
          data-tauri-drag-region
        />
        <Flex.Box
          pack
          x
          className={CSS(
            CSS.BE("login", "container"),
            servingCluster != null && CSS.M("narrow"),
          )}
          grow={false}
          rounded={1.5}
          background={0}
        >
          {servingCluster == null && (
            <Cluster.List
              className={CSS.BE("login", "list")}
              value={selectedKey}
              onChange={handleSelectedClusterChange}
            />
          )}
          <Flex.Box
            y
            gap="huge"
            className={CSS.BE("login", "form")}
            bordered
            grow
            shrink={false}
          >
            <Form.Form<typeof credentialsZ> {...methods}>
              <Flex.Box y align="center" grow gap="huge" shrink={false}>
                <Text.Text level="h2" color={11} weight={450}>
                  Log In
                </Text.Text>
                <Flex.Box y full="x" empty>
                  <Form.TextField path="username" inputProps={USERNAME_INPUT_PROPS} />
                  <Form.TextField path="password" inputProps={PASSWORD_INPUT_PROPS} />
                </Flex.Box>
                <Button.Button
                  onClick={handleSubmit}
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
    </Flex.Box>
  );
};
