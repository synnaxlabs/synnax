import { ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Divider,
  Form,
  Input,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
  usePrevious,
} from "@synnaxlabs/pluto";
import { change, deep } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { FC, ReactElement, useEffect, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { CSS } from "@/css";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { overviewLayout } from "@/range/external";
import { useSelect } from "@/range/selectors";
import { add, StaticRange } from "@/range/slice";

interface ParentRangeButtonProps {
  rangeKey: string;
}

const ParentRangeButton = ({
  rangeKey,
}: ParentRangeButtonProps): ReactElement | null => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const [parent, setParent] = useState<ranger.Range | null>();
  const placer = Layout.usePlacer();

  useAsyncEffect(async () => {
    try {
      if (client == null) return;
      const rng = await client.ranges.retrieve(rangeKey);
      const childRanges = await rng.retrieveParent();
      setParent(childRanges);
      const tracker = await rng.openParentRangeTracker();
      if (tracker == null) return;
      tracker.onChange((ranges) => setParent(ranges));
      return async () => await tracker.close();
    } catch (e) {
      addStatus({
        variant: "error",
        message: `Failed to retrieve child ranges`,
        description: (e as Error).message,
      });
      return undefined;
    }
  }, [rangeKey, client?.key]);
  if (parent == null) return null;
  return (
    <Align.Space direction="x" size="small" align="center">
      <Text.Text level="p">Child Range of</Text.Text>
      <Button.Button
        variant="text"
        shade={7}
        weight={400}
        startIcon={<Icon.Range />}
        iconSpacing="small"
        style={{ padding: "1rem" }}
        onClick={() =>
          placer({ ...overviewLayout, key: parent.key, name: parent.name })
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
    applyObservable: async ({ changes, ctx }) => {
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
  const name = Form.useFieldValue<string>("name", false, formCtx);
  const handleLink = Link.useCopyToClipboard();
  const handleCopyLink = () => {
    handleLink({ name, ontologyID: { key: rangeKey, type: "range" } });
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
      rng = client.ranges.retrieve("${rangeKey}")
    `,
      `Python code for ${name}`,
    );
  };

  const handleCopyTypeScriptCode = () => {
    const name = formCtx.get<string>("name").value;
    copy(
      `
      // Retrieve ${name}
      const rng = await client.ranges.retrieve("${rangeKey}")
    `,
      `TypeScript code for ${name}`,
    );
  };

  return (
    <Form.Form {...formCtx}>
      <Align.Space direction="x" justify="spaceBetween" className={CSS.B("header")}>
        <Align.Space direction="y" grow>
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
          direction="x"
          className={CSS.B("copy-buttons")}
          style={{ height: "fit-content" }}
          size="small"
        >
          <Align.Space direction="x">
            <Button.Icon
              tooltip={`Copy Python code for ${name}`}
              tooltipLocation="bottom"
              variant="text"
            >
              <Icon.Python
                onClick={handleCopyPythonCode}
                style={{ color: "var(--pluto-gray-l7)" }}
              />
            </Button.Icon>
            <Button.Icon
              variant="text"
              tooltip={`Copy TypeScript code for ${name}`}
              tooltipLocation="bottom"
              onClick={handleCopyTypeScriptCode}
            >
              <Icon.TypeScript style={{ color: "var(--pluto-gray-l7)" }} />
            </Button.Icon>
          </Align.Space>
          <Divider.Divider direction="y" />
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
      <Align.Space
        className={CSS.B("time-range")}
        direction="x"
        size="medium"
        align="center"
      >
        <Form.Field<number> path="timeRange.start" padHelpText={false} label="From">
          {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
        </Form.Field>
        <Text.WithIcon
          className={CSS.B("time-range-divider")}
          level="h4"
          startIcon={<Icon.Arrow.Right />}
        />
        <Form.Field<number> padHelpText={false} path="timeRange.end" label="To">
          {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
        </Form.Field>
      </Align.Space>
    </Form.Form>
  );
};
