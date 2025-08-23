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
  Button,
  Divider,
  Flex,
  Form,
  Icon,
  Input,
  Ranger,
  Status,
  Text,
  usePrevious,
} from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC, type ReactElement, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { CSS } from "@/css";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Label } from "@/label";
import { Layout } from "@/layout";
import { rename } from "@/layout/slice";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

interface ParentRangeButtonProps {
  rangeKey: string;
}

const ParentRangeButton = ({
  rangeKey,
}: ParentRangeButtonProps): ReactElement | null => {
  const res = Ranger.retrieveParent.useDirect({
    params: { id: ranger.ontologyID(rangeKey) },
  });
  const placeLayout = Layout.usePlacer();
  if (res.variant !== "success" || res.data == null) return null;
  const parent = res.data;
  return (
    <Flex.Box x gap="small" align="center">
      <Text.Text weight={450} color={9}>
        Child Range of
      </Text.Text>
      <Button.Button
        variant="text"
        weight={400}
        gap="small"
        style={{ padding: "1rem" }}
        onClick={() =>
          placeLayout({ ...OVERVIEW_LAYOUT, key: parent.key, name: parent.name })
        }
      >
        <Icon.Range />
        {parent.name}
      </Button.Button>
    </Flex.Box>
  );
};

export interface DetailsProps {
  rangeKey: string;
}

export const Details: FC<DetailsProps> = ({ rangeKey }) => {
  const layoutName = Layout.useSelect(rangeKey)?.name;
  const prevLayoutName = usePrevious(layoutName);
  const dispatch = useDispatch();
  const { form, status } = Ranger.useForm({
    params: { key: rangeKey },
    initialValues: {
      key: rangeKey,
      name: "",
      timeRange: { start: 0, end: 0 },
      labels: [],
    },
    autoSave: true,
  });
  const name = Form.useFieldValue<string, string, typeof Ranger.formSchema>("name", {
    ctx: form,
  });
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleCopyLink = () =>
    handleLink({ name, ontologyID: ranger.ontologyID(rangeKey) });

  useEffect(() => {
    if (
      prevLayoutName == layoutName ||
      prevLayoutName == null ||
      status.variant !== "success"
    )
      return;
    form.set("name", layoutName);
  }, [layoutName, status]);
  useEffect(() => {
    if (primitive.isNonZero(name)) dispatch(rename({ key: rangeKey, name }));
  }, [name]);

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

  if (status.variant === "error")
    return (
      <Status.Summary
        variant={status.variant}
        message={status.message}
        description={status.description}
      />
    );

  return (
    <Form.Form<typeof Ranger.formSchema> {...form}>
      <Flex.Box y gap="large">
        <Flex.Box x justify="between" className={CSS.B("header")}>
          <Flex.Box y grow>
            <Form.TextField
              path="name"
              showLabel={false}
              inputProps={{
                variant: "text",
                level: "h1",
                placeholder: "Name",
                onlyChangeOnBlur: true,
                resetOnBlurIfEmpty: true,
              }}
              padHelpText={false}
            />
            <ParentRangeButton rangeKey={rangeKey} />
          </Flex.Box>
          <Flex.Box
            x
            className={CSS.B("copy-buttons")}
            style={{ height: "fit-content" }}
            gap="small"
          >
            <Flex.Box x gap="small">
              <Button.Button
                tooltip={`Copy Python code to retrieve ${name}`}
                tooltipLocation="bottom"
                variant="text"
                onClick={handleCopyPythonCode}
              >
                <Icon.Python color={9} />
              </Button.Button>
              <Button.Button
                variant="text"
                tooltip={`Copy TypeScript code to retrieve ${name}`}
                tooltipLocation="bottom"
                onClick={handleCopyTypeScriptCode}
              >
                <Icon.TypeScript color={9} />
              </Button.Button>
            </Flex.Box>
            <Divider.Divider y />
            <Button.Button
              variant="text"
              tooltip={`Copy link to ${name}`}
              tooltipLocation="bottom"
              onClick={handleCopyLink}
            >
              <Icon.Link color={10} />
            </Button.Button>
          </Flex.Box>
        </Flex.Box>
        <Flex.Box className={CSS.B("time-range")} x gap="medium" align="center">
          <Form.Field<number> path="timeRange.start" padHelpText={false} label="From">
            {(p) => (
              <Input.DateTime level="h4" variant="text" onlyChangeOnBlur {...p} />
            )}
          </Form.Field>
          <Text.Text className={CSS.B("time-range-divider")} level="h4">
            <Icon.Arrow.Right />
          </Text.Text>
          <Form.Field<number> padHelpText={false} path="timeRange.end" label="To">
            {(p) => (
              <Input.DateTime onlyChangeOnBlur level="h4" variant="text" {...p} />
            )}
          </Form.Field>
        </Flex.Box>
        <Form.Field<string[]> required={false} path="labels">
          {({ variant: _, value, onChange }) => (
            <Label.SelectMultiple
              zIndex={100}
              variant="floating"
              location="bottom"
              style={{ width: "fit-content" }}
              value={value}
              onChange={onChange}
            />
          )}
        </Form.Field>
      </Flex.Box>
    </Form.Form>
  );
};
