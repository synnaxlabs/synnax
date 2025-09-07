import { type arc, DisconnectedError, NotFoundError } from "@synnaxlabs/client";
import { Button, Flex, Icon, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useCallback, useState } from "react";
import { useStore } from "react-redux";

import { Arc } from "@/arc";
import { translateSlateBackward } from "@/arc/types/translate";
import { useSelect } from "@/effect/selectors";
import { Layout } from "@/layout";
import { type RootState } from "@/store";

export interface LoadedProps {
  effect: arc.Effect;
  layoutKey: string;
}

const Loaded = ({ effect, layoutKey }: LoadedProps): ReactElement => {
  const client = Synnax.use();
  const store = useStore<RootState>();
  const layout = Layout.useSelect(layoutKey);
  const addStatus = Status.useAdder();
  const publishMut = useMutation({
    mutationFn: async () => {
      try {
        if (client == null) throw new DisconnectedError();
        const arc = Arc.select(store.getState(), effect.arc);
        await client.arcs.create(translateSlateBackward(arc));
        await client.effects.create({
          ...effect,
          name: layout?.name ?? "",
        });
      } catch (e) {
        console.log(e);
        addStatus(status.fromException(e));
      }
    },
  });

  const validate = useCallback(
    async (graph: arc.Graph) => {
      if (client == null) return null;
      try {
        await client.effects.validate(graph);
        return null;
      } catch (e) {
        return e as Error;
      }
    },
    [client],
  );
  return (
    <Flex.Box y grow style={{ height: "100%" }}>
      <Arc.Arc
        layoutKey={effect.arc}
        visible
        validate={validate}
        focused={false}
        onClose={() => {}}
      />
      <Flex.Box x style={{ padding: "2rem" }} justify="end" grow>
        <Flex.Box
          x
          background={1}
          style={{ padding: "2rem" }}
          bordered
          borderColor={5}
          grow
          rounded={2}
          justify="between"
        >
          <EffectState effect={effect} />
          <Button.Button onClick={() => publishMut.mutate()}>
            <Icon.Play />
            Deploy
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

const EffectState = ({ effect }: { effect: effect.Effect }) => {
  const client = Synnax.use();
  const [status, setStatus] = useState<effect.Status | null>(null);
  const addStatus = Status.useAdder();
  return (
    <Text.Text variant={status?.variant}>
      {status?.message ?? "Effect has not been deployed yet."}
    </Text.Text>
  );
};

export const Edit: Layout.Renderer = ({ layoutKey }) => {
  const client = Synnax.use();
  const effect = useSelect(layoutKey);
  const res = useQuery({
    queryKey: ["effect", layoutKey],
    queryFn: async () => {
      if (effect != null || client == null) return effect;
      try {
        const effect = await client.effects.retrieve({ key: layoutKey });
        return effect;
      } catch (e) {
        if (NotFoundError.matches(e)) return effect;
        throw e;
      }
    },
  });
  if (res.isLoading) return <Text.Text level="p">Loading...</Text.Text>;
  if (res.isError)
    return (
      <Text.Text center status="error">
        {res.error.message}
      </Text.Text>
    );
  return <Loaded effect={res.data as effect.Effect} layoutKey={layoutKey} />;
};
