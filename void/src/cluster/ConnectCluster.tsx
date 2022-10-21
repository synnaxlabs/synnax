import CoreWindow from "../lib/Windows/CoreWindow";
import { Dispatch, AnyAction } from "@reduxjs/toolkit";
import { createWindow } from "@synnaxlabs/drift";
import { Button, Header, Input, Nav, Space } from "@synnaxlabs/pluto";
import { AiFillApi, AiOutlineCheck } from "react-icons/ai";
import { FieldValues, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Connectivity,
  Synnax,
  SynnaxProps,
  synnaxPropsSchema,
} from "@synnaxlabs/client";
import "./ConnectCluster.css";
import { useState } from "react";
import { ConnectionState, setCluster } from "./slice";
import ConnectionStatus from "./ConnectionStatus";
import { useSelector, useDispatch } from "react-redux";
import { closeWindow } from "@synnaxlabs/drift";
import { z } from "zod";

const formSchema = synnaxPropsSchema.extend({
  name: z.string().optional(),
});

export default function ConnectCluster() {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({ resolver: zodResolver(formSchema) });

  const dispatch = useDispatch();

  const [connectionState, setConnectionState] = useState<
    ConnectionState | undefined
  >(undefined);

  const onSubmit = (data: FieldValues) => {
    const client = new Synnax(data as SynnaxProps);
    setConnectionState({
      status: Connectivity.CONNECTING,
      message: "Connecting...",
    });
    client.connectivity
      .check()
      .then(() => {
        const status = client.connectivity.status();
        if (status === Connectivity.CONNECTED) {
          const state = { status, message: "Connected" };
          setConnectionState({ status, message: "Connected" });
          dispatch(
            setCluster({
              active: true,
              name: data.name,
              props: data as SynnaxProps,
              state,
            })
          );
        } else {
          setConnectionState({
            status,
            message: client.connectivity.error()?.message || "",
          });
        }
      })
      .catch((err) => {
        setConnectionState({
          status: Connectivity.FAILED,
          message: err.message,
        });
      });
  };

  return (
    <CoreWindow>
      <Header level="h4" icon={<AiFillApi />} divided>
        Connect a Cluster
      </Header>
      <Space className="connect-cluster__content" direction="vertical" grow>
        <form onSubmit={handleSubmit(onSubmit)} id="my-form">
          <Space direction="vertical">
            <Input.Item
              label="Name"
              placeholder="My Synnax Cluster"
              {...register("name")}
            />
            <Space direction="horizontal">
              <Input.Item
                label="Host"
                placeholder="localhost"
                helpText={errors.host?.message?.toString()}
                style={{ flexGrow: 1 }}
                {...register("host")}
              />
              <Input.Item
                label="Port"
                type="number"
                placeholder="8080"
                helpText={errors.port?.message?.toString()}
                style={{ width: 100 }}
                {...register("port")}
              />
            </Space>
            <Input.Item
              label="Username"
              placeholder="Harry"
              helpText={errors.username?.message?.toString()}
              {...register("username")}
            />
            <Input.Item
              label="Password"
              placeholder="Seldon"
              type="password"
              helpText={errors.password?.message?.toString()}
              {...register("password")}
            />
          </Space>
        </form>
      </Space>
      <Nav.Bar location="bottom" size={48} style={{ flexShrink: 0 }}>
        <Nav.Bar.Start style={{ padding: "0 12px" }}>
          {connectionState && <ConnectionStatus state={connectionState} />}
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ padding: 6 }}>
          {connectionState?.status === Connectivity.CONNECTED ? (
            <Button variant="filled" onClick={() => dispatch(closeWindow())}>
              Done
            </Button>
          ) : (
            <Button type="submit" size="medium" form="my-form">
              Connect
            </Button>
          )}
        </Nav.Bar.End>
      </Nav.Bar>
    </CoreWindow>
  );
}

export const createConnectClusterWindow = (dispatch: Dispatch<AnyAction>) => {
  dispatch(
    createWindow({
      url: "http://localhost:5173/cluster/connect",
      resizable: false,
      height: 425,
      width: 550,
      title: "Connect a cluster",
    })
  );
};
