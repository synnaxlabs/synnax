import { Button, Header, Input, Nav, Space } from "@synnaxlabs/pluto";
import { AiFillApi } from "react-icons/ai";
import { FieldValues, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Connectivity, SynnaxProps, synnaxPropsSchema } from "@synnaxlabs/client";
import "./ConnectCluster.css";
import { useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { ConnectionState } from "@/features/cluster/types";
import { testConnection } from "../util/testConnection";
import { setActiveCluster, setCluster, useSelectCluster } from "../store";
import { LayoutRendererProps, useSelectLayout } from "@/features/layout";
import { ConnectionStateBadge } from "./ConnectionStateBadge";

const formSchema = synnaxPropsSchema.extend({
	name: z.string().optional(),
});

export interface ConnectClusterContentProps {
	clusterKey?: string;
}

export const ConnectCluster = ({ layoutKey, onClose }: LayoutRendererProps) => {
	const dispatch = useDispatch();
	const [connState, setConnState] = useState<ConnectionState | null>(null);

	const {
		getValues,
		trigger,
		register,
		handleSubmit,
		formState: { errors },
	} = useForm({
		resolver: zodResolver(formSchema),
	});

	const onSubmit = async (data: FieldValues) => {
		const name = data.name;
		delete data.name;
		const { clusterKey, state } = await testConnection(data as SynnaxProps);
		if (state.status !== Connectivity.Connected) return setConnState(state);
		dispatch(
			setCluster({
				key: clusterKey as string,
				name: name,
				state: state,
				props: data as SynnaxProps,
			})
		);
		dispatch(setActiveCluster(clusterKey as string));
		onClose();
	};

	const handleTestConnection = async () => {
		const ok = await trigger();
		if (!ok) return;
		const { state } = await testConnection(getValues() as SynnaxProps);
		setConnState(state);
	};

	return (
		<Space direction="vertical" grow>
			<Header level="h4" icon={<AiFillApi />} divided>
				Connect a Cluster
			</Header>
			<Space className="connect-cluster__content" direction="vertical" grow>
				<form onSubmit={handleSubmit(onSubmit)} id="connect-cluster">
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
								className="connect-cluster__input__host"
								{...register("host")}
							/>
							<Input.Item
								label="Port"
								type="number"
								placeholder="8080"
								helpText={errors.port?.message?.toString()}
								className="connect-cluster__input__port"
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
			<Nav.Bar location="bottom" size={48}>
				<Nav.Bar.Start style={{ padding: "0 2rem" }}>
					{connState && <ConnectionStateBadge state={connState} />}
				</Nav.Bar.Start>
				<Nav.Bar.End style={{ padding: "1rem" }}>
					<Button variant="text" size="medium" onClick={handleTestConnection}>
						Test Connection
					</Button>
					<Button variant="filled" type="submit" form="connect-cluster">
						Done
					</Button>
				</Nav.Bar.End>
			</Nav.Bar>
		</Space>
	);
};
