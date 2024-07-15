// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { List, Synnax, Text } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";

export interface RackOverviewProps {
  rackKey: string;
}

export interface TaskListProps {
  rackKey: number;
}

export const TaskList = ({ rackKey }: TaskListProps): ReactElement => {
  const client = Synnax.use();

  const { data } = useQuery({
    queryKey: ["tasks", rackKey],
    queryFn: async () => await client?.hardware.tasks.retrieve(rackKey),
  });

  return (
    <List.List<string, task.Task> data={data}>
      <List.Core<string, task.Task>>{(p) => <TaskListItem {...p} />}</List.Core>
    </List.List>
  );
};

interface TaskListItemProps extends List.ItemProps<string, task.Task> {}

const TaskListItem = (props: TaskListItemProps): ReactElement => {
  const {
    entry: { name },
  } = props;
  return (
    <List.ItemFrame {...props}>
      <Text.Text level="p">{name}</Text.Text>
    </List.ItemFrame>
  );
};
