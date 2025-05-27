import { type effect, NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";
import { useStore } from "react-redux";
import { v4 as uuid } from "uuid";

import { useSelect } from "@/effect/selectors";
import { NULL_CLIENT_ERROR } from "@/errors";
import { type Layout } from "@/layout";
import { Slate } from "@/slate";
import { translateSlateBackward } from "@/slate/types/translate";
import { type RootState } from "@/store";

export interface LoadedProps {
  effect: effect.Effect;
}

const Loaded = ({ effect }: LoadedProps): ReactElement => {
  const client = Synnax.use();
  const store = useStore<RootState>();
  const publishMut = useMutation({
    mutationFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const slate = Slate.select(store.getState(), effect.slate);
      await client.slates.create(translateSlateBackward(slate));
      await client.effects.create(effect);
    },
  });
  return (
    <Align.Space y grow style={{ height: "100%" }}>
      <Slate.Slate
        layoutKey={effect.slate}
        visible
        focused={false}
        onClose={() => {}}
      />
      <Align.Space x background={2} style={{ padding: "2rem" }} bordered justify="end">
        <Button.Button startIcon={<Icon.Play />} onClick={() => publishMut.mutate()}>
          Publish
        </Button.Button>
      </Align.Space>
    </Align.Space>
  );
};

export const Edit: Layout.Renderer = ({ layoutKey }) => {
  const client = Synnax.use();
  const effect = useSelect(layoutKey);
  const res = useQuery({
    queryKey: ["effect", layoutKey],
    queryFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (effect != null) return effect;
      try {
        const effect = await client.effects.retrieve(layoutKey);
        return effect;
      } catch (e) {
        if (NotFoundError.matches(e)) return effect;
        throw e;
      }
    },
  });
  console.log(res.data);
  if (res.isLoading) return <Text.Text level="p">Loading...</Text.Text>;
  if (res.isError)
    return (
      <Align.Space y grow style={{ height: "100%" }}>
        <Status.Text.Centered variant="error">{res.error.message}</Status.Text.Centered>
      </Align.Space>
    );
  return <Loaded effect={res.data as effect.Effect} />;
};
