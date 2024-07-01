import { task, UnexpectedError } from "@synnaxlabs/client";
import { Align, Status, Synnax } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useQuery } from "@tanstack/react-query";
import { FC } from "react";

import { Layout } from "@/layout";

interface WrappedProps<T extends task.Task, P extends task.Payload> {
  layoutKey: string;
  task?: T;
  initialValues: P;
}

export const wrapTaskLayout = <T extends task.Task, P extends task.Payload>(
  Wrapped: FC<WrappedProps<T, P>>,
  zeroPayload: P,
): Layout.Renderer => {
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const client = Synnax.use();
    const args = Layout.useSelectArgs<{ create: boolean }>(layoutKey);
    const altKey = Layout.useSelectAltKey(layoutKey);
    const fetchTask = useQuery<WrappedProps<T, P>>({
      queryKey: [layoutKey, client?.key, altKey],
      queryFn: async () => {
        if (client == null || args.create)
          return { initialValues: deep.copy(zeroPayload), layoutKey };
        // try to parse the key as a big int. If the parse fails, set the lat key as a key
        let key: string = layoutKey;
        try {
          BigInt(layoutKey);
        } catch (e) {
          if (altKey == undefined)
            throw new UnexpectedError(
              `Task has non-bigint layout key ${layoutKey} with no alternate key`,
            );
          if (e instanceof SyntaxError) key = altKey;
        }
        const t = await client.hardware.tasks.retrieve(key, { includeState: true });
        return { initialValues: t as unknown as P, task: t as T, layoutKey };
      },
    });
    if (fetchTask.isPending) return <></>;
    if (fetchTask.isError)
      return (
        <Align.Space direction="y" grow style={{ height: "100%" }}>
          <Status.Text.Centered variant="error">
            {fetchTask.error.message}
          </Status.Text.Centered>
        </Align.Space>
      );
    return <Wrapped {...fetchTask.data} layoutKey={layoutKey} />;
  };
  Wrapper.displayName = `TaskWrapper(${Wrapped.displayName ?? Wrapped.name})`;
  return Wrapper;
};
