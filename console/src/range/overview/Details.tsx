// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Divider,
  Form,
  Icon,
  Input,
  Ranger,
  Text,
  usePrevious,
} from "@synnaxlabs/pluto";
import { type change, deep } from "@synnaxlabs/x";
import { type FC, type ReactElement, useEffect } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { Cluster } from "@/cluster";
import { CSS } from "@/css";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Layout } from "@/layout";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";
import { useSelect } from "@/range/selectors";
import { add, type StaticRange } from "@/range/slice";

interface ParentRangeButtonProps {
  rangeKey: string;
}

const ParentRangeButton = ({
  rangeKey,
}: ParentRangeButtonProps): ReactElement | null => {
  const parent = Ranger.useRetrieveParentRange(rangeKey);
  const placeLayout = Layout.usePlacer();

  if (parent == null) return null;
  return (
    <Align.Space x gap="small" align="center">
      <Text.Text level="p" shade={11} weight={450}>
        Child Range of
      </Text.Text>
      <Button.Button
        variant="text"
        shade={11}
        weight={400}
        startIcon={<Icon.Range />}
        gap="small"
        style={{ padding: "1rem" }}
        onClick={() =>
          placeLayout({ ...OVERVIEW_LAYOUT, key: parent.key, name: parent.name })
        }
      >
        {parent.name}
      </Button.Button>
    </Align.Space>
  );
};

export interface DetailsProps {
  rangeKey: string;
}

const formSchema = z.object({
  name: z.string().min(1, "Name must not be empty"),
  timeRange: z.object({
    start: z.number(),
    end: z.number(),
  }),
});

export const Details: FC<DetailsProps> = ({ rangeKey }) => {
  const existingRangeInState = useSelect(rangeKey);
  const layoutName = Layout.useSelect(rangeKey)?.name;
  const prevLayoutName = usePrevious(layoutName);
  const dispatch = useDispatch();

  const formCtx = Form.useSynced<
    typeof formSchema,
    change.Change<string, ranger.Range>[]
  >({
    name: "Range",
    key: [rangeKey, "details"],
    schema: formSchema,
    values: {
      name: "",
      timeRange: { start: 0, end: 0 },
    },
    queryFn: async ({ client }) => {
      const rng = await client.ranges.retrieve(rangeKey);
      return {
        name: rng.name,
        timeRange: {
          start: Number(rng.timeRange.start),
          end: Number(rng.timeRange.end),
        },
      };
    },
    openObservable: async (client) => await client.ranges.openTracker(),
    applyObservable: ({ changes, ctx }) => {
      const target = changes.find((c) => c.variant === "set" && c.key === rangeKey);
      if (target == null || target.value == null) return;
      ctx.set("", {
        name: target.value.name,
        timeRange: {
          start: Number(target.value.timeRange.start),
          end: Number(target.value.timeRange.end),
        },
      });
    },
    applyChanges: async ({ client, path, values, prev }) => {
      if (client == null || deep.equal(values, prev)) return;
      const { name, timeRange } = values;
      await client.ranges.create({ key: rangeKey, name, timeRange });
      if (existingRangeInState == null) return;
      if (path.includes("name")) dispatch(Layout.rename({ key: rangeKey, name }));
      const newRange: StaticRange = {
        key: rangeKey,
        persisted: true,
        variant: "static",
        name,
        timeRange: {
          start: Number(timeRange.start),
          end: Number(timeRange.end),
        },
      };
      dispatch(add({ ranges: [newRange], switchActive: false }));
    },
  });
  const name = Form.useFieldValue<string, string, typeof formSchema>("name", {
    ctx: formCtx,
  });
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleCopyLink = () => {
    handleLink({ name, ontologyID: ranger.ontologyID(rangeKey) });
  };

  useEffect(() => {
    if (prevLayoutName == layoutName || prevLayoutName == null) return;
    formCtx.set("name", layoutName);
  }, [layoutName]);

  const copy = useCopyToClipboard();
  const handleCopyPythonCode = () => {
    copy(
      `
      # Retrieve ${name}
      rng = client.ranges.retrieve(key="${rangeKey}")
    `,
      `Python code to retrieve ${name}`,
    );
  };

  const handleCopyTypeScriptCode = () => {
    const name = formCtx.get<string>("name").value;
    copy(
      `
      // Retrieve ${name}
      const rng = await client.ranges.retrieve("${rangeKey}")
    `,
      `TypeScript code to retrieve ${name}`,
    );
  };

  return (
    <Form.Form<typeof formSchema> {...formCtx}>
      <Align.Space y gap="large">
        <Align.Space x justify="spaceBetween" className={CSS.B("header")}>
          <Align.Space y grow>
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
            <ParentRangeButton rangeKey={rangeKey} />
          </Align.Space>
          <Align.Space
            x
            className={CSS.B("copy-buttons")}
            style={{ height: "fit-content" }}
            gap="small"
          >
            <Align.Space x>
              <Button.Icon
                tooltip={`Copy Python code to retrieve ${name}`}
                tooltipLocation="bottom"
                variant="text"
              >
                <Icon.Python
                  onClick={handleCopyPythonCode}
                  style={{ color: "var(--pluto-gray-l9)" }}
                />
              </Button.Icon>
              <Button.Icon
                variant="text"
                tooltip={`Copy TypeScript code to retrieve ${name}`}
                tooltipLocation="bottom"
                onClick={handleCopyTypeScriptCode}
              >
                <Icon.TypeScript style={{ color: "var(--pluto-gray-l9)" }} />
              </Button.Icon>
            </Align.Space>
            <Divider.Divider y />
            <Button.Icon
              variant="text"
              tooltip={`Copy link to ${name}`}
              tooltipLocation="bottom"
              onClick={handleCopyLink}
            >
              <Icon.Link />
            </Button.Icon>
          </Align.Space>
        </Align.Space>
        <Align.Space className={CSS.B("time-range")} x gap="medium" align="center">
          <Form.Field<number> path="timeRange.start" padHelpText={false} label="From">
            {(p) => (
              <Input.DateTime level="h4" variant="natural" onlyChangeOnBlur {...p} />
            )}
          </Form.Field>
          <Text.WithIcon
            className={CSS.B("time-range-divider")}
            level="h4"
            startIcon={<Icon.Arrow.Right />}
          />
          <Form.Field<number> padHelpText={false} path="timeRange.end" label="To">
            {(p) => (
              <Input.DateTime onlyChangeOnBlur level="h4" variant="natural" {...p} />
            )}
          </Form.Field>
        </Align.Space>
      </Align.Space>
    </Form.Form>
  );
};
