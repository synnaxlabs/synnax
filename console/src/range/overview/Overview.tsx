// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/overview/Overview.css";

import { ontology, ranger, TimeRange } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Divider,
  Form,
  Input,
  Label,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ReactElement, useCallback } from "react";
import { z } from "zod";

import { Layout } from "@/layout";
import { Link } from "@/link";
import { updateLabels } from "@/range/EditLayout";
import { MetaData } from "@/range/overview/MetaData";
import { SubRanges } from "@/range/overview/SubRanges";

export const OVERVIEW_TYPE = "overview";

export const overviewLayout: Layout.State = {
  key: "overview",
  windowKey: "overview",
  type: "overview",
  name: "Overview",
  location: "mosaic",
  tab: {
    editable: false,
  },
};

const formSchema = z.object({
  name: z.string().min(1, "Name must not be empty"),
  timeRange: TimeRange.z,
  labels: z.array(z.string()),
});

export const Overview: Layout.Renderer = ({ layoutKey }): ReactElement => {
  const client = Synnax.use();

  const rng = useQuery({
    queryKey: ["range", layoutKey],
    queryFn: async () => {
      if (client == null) return;
      const rng = await client.ranges.retrieve(layoutKey);
      const labels = await client.labels.retrieveFor(
        new ontology.ID({ key: layoutKey, type: "range" }),
      );
      return {
        rng,
        layoutKey,
        initialValues: {
          name: rng.name,
          timeRange: {
            start: Number(rng.timeRange.start),
            end: Number(rng.timeRange.end),
          },
          labels: labels.map((l) => l.key),
        },
      };
    },
  });
  if (rng.isPending || rng.isError) return <div>Loading...</div>;
  return <Internal layoutKey={layoutKey} {...rng.data} />;
};

interface InternalProps {
  layoutKey: string;
  rng: ranger.Range;
  initialValues: z.output<typeof formSchema>;
}

interface ParentRangeButtonProps {
  rng: ranger.Range;
}

const ParentRangeButton = ({ rng }: ParentRangeButtonProps): ReactElement | null => {
  const client = Synnax.use();
  const parent = useQuery({
    queryKey: [rng.key, "parent"],
    queryFn: async () => {
      if (client == null) return;
      return await rng.retrieveParent();
    },
  });
  const placer = Layout.usePlacer();
  if (parent.isPending || parent.data == null) return null;
  return (
    <Align.Space direction="x" size="small" align="center">
      <Text.Text level="p">Sub-range of</Text.Text>
      <Button.Button
        variant="text"
        shade={7}
        weight={400}
        startIcon={<Icon.Range />}
        iconSpacing="small"
        style={{ padding: "1rem" }}
        onClick={() =>
          placer({ ...overviewLayout, key: parent.data?.key, name: parent.data?.name })
        }
      >
        {parent.data?.name}
      </Button.Button>
    </Align.Space>
  );
};

const Internal = ({ rng, initialValues }: InternalProps): ReactElement => {
  const client = Synnax.use();

  const updateRange = useMutation({
    mutationFn: async ({
      values,
      path,
      prev,
    }: Form.OnChangeProps<typeof formSchema>) => {
      if (client == null) return;
      if (path === "labels") {
        const { labels } = values;
        const prevL = prev as string[];
        await updateLabels(client, rng.key, prevL, labels);
        return;
      }
      const { name, timeRange } = values;
      await client.ranges.create({
        key: rng.key,
        name,
        timeRange: {
          start: timeRange.start,
          end: timeRange.end,
        },
      });
    },
  });

  const methods = Form.use({
    values: initialValues,
    schema: formSchema,
    onChange: updateRange.mutate,
  });

  const handleLink = Link.useCopyToClipboard();
  const handleCopyLink = useCallback(() => {
    handleLink({
      name: rng.name,
      ontologyID: { key: rng.key, type: "range" },
    });
  }, [rng, handleLink]);

  return (
    <Align.Space
      direction="y"
      style={{ padding: "5rem", maxWidth: "1200px", margin: "0 auto" }}
    >
      <Form.Form {...methods}>
        <Align.Space direction="x" justify="spaceBetween">
          <Align.Space direction="y">
            <Form.TextField
              path="name"
              showLabel={false}
              inputProps={{
                variant: "natural",
                level: "h1",
                placeholder: "Name",
                onlyChangeOnBlur: true,
                resetOnBlurIfEmpty: true,
              }}
              padHelpText={false}
            />
            <ParentRangeButton rng={rng} />
          </Align.Space>
          <Align.Space direction="x" style={{ height: "fit-content" }} size="small">
            <Align.Space direction="x">
              <Button.Icon variant="text">
                <Icon.Python style={{ color: "var(--pluto-gray-l7)" }} />
              </Button.Icon>
              <Button.Icon>
                <Icon.TypeScript style={{ color: "var(--pluto-gray-l7)" }} />
              </Button.Icon>
            </Align.Space>
            <Divider.Divider direction="y" />
            <Button.Icon variant="text" onClick={handleCopyLink}>
              <Icon.Link />
            </Button.Icon>
          </Align.Space>
        </Align.Space>
        <Align.Space direction="x" size="large">
          <Form.Field<number> path="timeRange.start" label="From">
            {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
          </Form.Field>
          <Text.WithIcon level="h4" startIcon={<Icon.Arrow.Right />} />
          <Form.Field<number> path="timeRange.end" label="To">
            {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
          </Form.Field>
        </Align.Space>
        <Form.Field<string> required={false} path="labels">
          {(p) => (
            <Label.SelectMultiple
              searcher={client?.labels}
              entryRenderKey="name"
              dropdownVariant="floating"
              zIndex={100}
              location="bottom"
              style={{ width: "fit-content" }}
              {...p}
            />
          )}
        </Form.Field>
      </Form.Form>
      <Divider.Divider direction="x" />
      <MetaData rng={rng} />
      <Divider.Divider direction="x" />
      <SubRanges rng={rng} />
    </Align.Space>
  );
};
