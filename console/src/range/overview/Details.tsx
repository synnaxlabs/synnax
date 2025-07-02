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
  Label,
  Ranger,
  Text,
  usePrevious,
} from "@synnaxlabs/pluto";
import { type FC, type ReactElement, useEffect } from "react";

import { Cluster } from "@/cluster";
import { CSS } from "@/css";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Layout } from "@/layout";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

interface ParentRangeButtonProps {
  rangeKey: string;
}

const ParentRangeButton = ({
  rangeKey,
}: ParentRangeButtonProps): ReactElement | null => {
  const res = Ranger.useParent(rangeKey);
  const placeLayout = Layout.usePlacer();
  if (res.variant !== "success" || res.data == null) return null;
  const parent = res.data;
  return (
    <Align.Space x size="small" align="center">
      <Text.Text level="p" shade={11} weight={450}>
        Child Range of
      </Text.Text>
      <Button.Button
        variant="text"
        shade={11}
        weight={400}
        startIcon={<Icon.Range />}
        iconSpacing="small"
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

export const Details: FC<DetailsProps> = ({ rangeKey }) => {
  const layoutName = Layout.useSelect(rangeKey)?.name;
  const prevLayoutName = usePrevious(layoutName);
  const { form } = Ranger.useForm({
    params: { key: rangeKey },
    initialValues: {
      key: rangeKey,
      name: "",
      timeRange: { start: 0, end: 0 },
      labels: [],
    },
  });
  const name = Form.useFieldValue<string, string, typeof Ranger.rangeFormSchema>(
    "name",
    { ctx: form },
  );
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleCopyLink = () => {
    handleLink({ name, ontologyID: ranger.ontologyID(rangeKey) });
  };

  useEffect(() => {
    if (prevLayoutName == layoutName || prevLayoutName == null) return;
    form.set("name", layoutName);
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
    const name = form.get<string>("name").value;
    copy(
      `
      // Retrieve ${name}
      const rng = await client.ranges.retrieve("${rangeKey}")
    `,
      `TypeScript code to retrieve ${name}`,
    );
  };

  return (
    <Form.Form<typeof Ranger.rangeFormSchema> {...form}>
      <Align.Space y size="large">
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
            size="small"
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
        <Align.Space className={CSS.B("time-range")} x size="medium" align="center">
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
          <Form.Field<string[]> required={false} path="labels">
            {({ variant: _, ...p }) => (
              <Label.SelectMultiple
                entryRenderKey="name"
                dropdownVariant="floating"
                zIndex={100}
                location="bottom"
                style={{ width: "fit-content" }}
                {...p}
              />
            )}
          </Form.Field>
        </Align.Space>
      </Align.Space>
    </Form.Form>
  );
};
