// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { synnaxPropsZ } from "@synnaxlabs/client";
import type { connection, SynnaxProps } from "@synnaxlabs/client";
import { Nav, componentRenderProp, Status } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Button } from "@synnaxlabs/pluto/button";
import { Input } from "@synnaxlabs/pluto/input";
import { Case } from "@synnaxlabs/x";
import { type FieldValues, useForm, FormProvider } from "react-hook-form";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { statusVariants } from "@/cluster/Badges";
import { useSelectMany } from "@/cluster/selectors";
import { setActive, set } from "@/cluster/slice";
import { testConnection } from "@/cluster/testConnection";
import { CSS } from "@/css";
import { type Layout } from "@/layout";

import "@/cluster/Connect.css";

export const connectWindowLayout: Layout.LayoutState = {
  key: "connectCluster",
  windowKey: "connectCluster",
  type: "connectCluster",
  name: "Connect Cluster",
  location: "window",
  window: {
    resizable: false,
    size: { height: 430, width: 650 },
    navTop: true,
    transparent: true,
  },
};

/**
 * ConnectCluster implements the LayoutRenderer component type to provide a form for
 * connecting to a cluster.
 *
 * @param props - The standard LayoutRendererProps.
 */
export const Connect = ({ onClose }: Layout.RendererProps): ReactElement => {
  const dispatch = useDispatch();
  const [connState, setConnState] = useState<connection.State | null>(null);
  const [loading, setLoading] = useState<"test" | "submit" | null>(null);

  const names = useSelectMany().map((c) => c.name);

  const formSchema = synnaxPropsZ.extend({
    name: z.string().refine((n) => !names.includes(n), {
      message: "A cluster with this name already exists",
    }),
  });

  const methods = useForm({ resolver: zodResolver(formSchema) });
  const { getValues, trigger, control: c, handleSubmit: _handleSubmit } = methods;
  const handleSubmit = _handleSubmit(async (_data: FieldValues): Promise<void> => {
    const { name, ...data } = _data;
    setConnState(null);
    setLoading("submit");
    const state = await testConnection(data as SynnaxProps);
    setLoading(null);
    if (state.status !== "connected") return setConnState(state);
    dispatch(
      set({
        key: state.clusterKey,
        name,
        props: data as SynnaxProps,
      }),
    );
    dispatch(setActive(state.clusterKey));
    onClose();
  });

  const handleTestConnection = (): void => {
    void (async (): Promise<void> => {
      const valid = await trigger();
      if (!valid) return;
      setConnState(null);
      setLoading("test");
      const state = await testConnection(getValues() as SynnaxProps);
      setConnState(state);
      setLoading(null);
    })();
  };

  return (
    <Align.Space grow className={CSS.B("connect-cluster")}>
      <FormProvider {...methods}>
        <Align.Space
          id="connect-cluster"
          el="form"
          /* eslint-disable-next-line @typescript-eslint/no-misused-promises */
          onSubmit={handleSubmit}
          className="console-form"
          grow
          empty
          justify="center"
        >
          <Input.HFItem<string> name="name">
            {(p) => <Input.Text placeholder="My Synnax Cluster" {...p} />}
          </Input.HFItem>
          <Align.Space direction="x">
            <Input.HFItem<string> name="host" grow>
              {(p) => <Input.Text placeholder="localhost" {...p} />}
            </Input.HFItem>
            <Input.HFItem<number> name="port" className={CSS.BE("input", "port")}>
              {(p) => <Input.Text placeholder="9090" {...p} />}
            </Input.HFItem>
          </Align.Space>
          <Input.HFItem<string> name="username">
            {(p) => <Input.Text placeholder="Harry" {...p} />}
          </Input.HFItem>
          <Align.Space direction="x">
            <Input.HFItem<string>
              name="password"
              className={CSS.BE("input", "password")}
            >
              {(p) => <Input.Text {...p} placeholder="Seldon" type="password" />}
            </Input.HFItem>
            <Input.HFItem<boolean> name="secure">
              {componentRenderProp(Input.Switch)}
            </Input.HFItem>
          </Align.Space>
        </Align.Space>
      </FormProvider>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start className={CSS.BE("footer", "start")}>
          {connState != null && (
            <Status.Text variant={statusVariants[connState.status]}>
              {connState.status === "connected"
                ? Case.capitalize(connState.status)
                : connState.message!}
            </Status.Text>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End className={CSS.BE("footer", "end")}>
          <Button.Button
            loading={loading === "test"}
            disabled={loading !== null}
            variant="text"
            onClick={handleTestConnection}
          >
            Test Connection
          </Button.Button>

          <Button.Button
            type="submit"
            form="connect-cluster"
            loading={loading === "submit"}
            disabled={loading !== null}
          >
            Done
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
