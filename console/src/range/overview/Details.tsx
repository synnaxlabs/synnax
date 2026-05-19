// Copyright 2026 Synnax Labs, Inc.
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
} from "@synnaxlabs/pluto";
import { type NumericTimeRange, TimeStamp } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback } from "react";

import { Cluster } from "@/cluster";
import { CSS } from "@/css";
import { CSV } from "@/csv";
import { Label } from "@/label";
import { Layout } from "@/layout";
import { FavoriteButton } from "@/range/FavoriteButton";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

interface ParentRangeButtonProps {
  rangeKey: string;
}

const ParentRangeButton = ({
  rangeKey,
}: ParentRangeButtonProps): ReactElement | null => {
  const res = Ranger.useRetrieveParent({ id: ranger.ontologyID(rangeKey) });
  const placeLayout = Layout.usePlacer();
  if (res.variant !== "success" || res.data == null) return null;
  const parent = res.data;
  const Icon = Ranger.STAGE_ICONS[Ranger.getStage(parent.timeRange)];
  return (
    <Flex.Box x gap="small" align="center">
      <Text.Text weight={450} color={9}>
        Child Range of
      </Text.Text>
      <Button.Button
        color={8}
        variant="text"
        weight={400}
        gap="small"
        style={{ padding: "1rem" }}
        onClick={() =>
          placeLayout({ ...OVERVIEW_LAYOUT, key: parent.key, name: parent.name })
        }
      >
        <Icon />
        {parent.name}
      </Button.Button>
    </Flex.Box>
  );
};

export interface DetailsProps {
  rangeKey: string;
}

export const Details: FC<DetailsProps> = ({ rangeKey }) => {
  const { data: range } = Ranger.useRetrieve({ key: rangeKey });
  const now = TimeStamp.now().nanoseconds;
  const { form, status } = Ranger.useForm({
    query: { key: rangeKey },
    initialValues: {
      key: rangeKey,
      name: "",
      timeRange: { start: now, end: now },
      labels: [],
    },
    autoSave: true,
  });

  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleError = Status.useErrorHandler();
  const name = Form.useFieldValue<string, string, typeof Ranger.formSchema>("name", {
    ctx: form,
  });
  const handleCopyLink = () =>
    handleLink({ name, ontologyID: ranger.ontologyID(rangeKey) });

  const getPythonCode = useCallback(
    () =>
      `
      # Retrieve ${name}
      rng = client.ranges.retrieve(key="${rangeKey}")
    `,
    [name, rangeKey],
  );
  const getTypeScriptCode = useCallback(
    () =>
      `
      // Retrieve ${name}
      const rng = await client.ranges.retrieve("${rangeKey}")
    `,
    [name, rangeKey],
  );

  const promptDownloadCSVModal = CSV.useDownloadModal();

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
          <Flex.Box x style={{ height: "fit-content" }} gap="small">
            <Button.Copy
              text={getPythonCode}
              tooltip={`Copy Python code to retrieve ${name}`}
              tooltipLocation="bottom"
              variant="text"
              successMessage={`Copied Python code to retrieve ${name} to clipboard`}
              textColor={9}
            >
              <Icon.Python />
            </Button.Copy>
            <Button.Copy
              variant="text"
              text={getTypeScriptCode}
              tooltip={`Copy TypeScript code to retrieve ${name}`}
              tooltipLocation="bottom"
              successMessage={`Copied TypeScript code to retrieve ${name} to clipboard`}
              textColor={9}
            >
              <Icon.TypeScript />
            </Button.Copy>
            <Divider.Divider y />
            <Button.Button
              tooltip={`Download data for ${name} as a CSV`}
              tooltipLocation="bottom"
              variant="text"
              textColor={9}
              onClick={() =>
                handleError(async () => {
                  await promptDownloadCSVModal(
                    {
                      timeRange: form.get<NumericTimeRange>("timeRange").value,
                      name,
                      channels: [],
                    },
                    { icon: "Range" },
                  );
                }, "Failed to download CSV")
              }
            >
              <Icon.CSV />
            </Button.Button>
            <Divider.Divider y />
            <Button.Button
              variant="text"
              tooltip={`Copy link to ${name}`}
              tooltipLocation="bottom"
              onClick={handleCopyLink}
              textColor={9}
            >
              <Icon.Link />
            </Button.Button>
            {range != null && <FavoriteButton range={range} size="medium" />}
          </Flex.Box>
        </Flex.Box>
        <Flex.Box className={CSS.B("time-range")} x gap="medium" align="center">
          <Form.Field<number> path="timeRange.start" padHelpText={false} label="From">
            {(p) => (
              <Input.DateTime level="h4" variant="text" onlyChangeOnBlur {...p} />
            )}
          </Form.Field>
          <Icon.Arrow.Right style={{ width: "3rem", height: "3rem" }} color={9} />
          <Form.Field<number> padHelpText={false} path="timeRange.end" label="To">
            {(p) => (
              <Input.DateTime onlyChangeOnBlur level="h4" variant="text" {...p} />
            )}
          </Form.Field>
        </Flex.Box>
        <Flex.Box x>
          <Form.Field<NumericTimeRange> path="timeRange" label="Stage">
            {(props) => (
              <Ranger.SelectStage
                {...Ranger.wrapNumericTimeRangeToStage(props)}
                allowNone={false}
                triggerProps={{ variant: "text", hideCaret: true }}
                variant="floating"
              />
            )}
          </Form.Field>
          <Form.Field<string[]> required={false} path="labels">
            {({ variant: _, ...p }) => (
              <Label.SelectMultiple
                zIndex={100}
                variant="floating"
                style={{ width: "fit-content" }}
                {...p}
              />
            )}
          </Form.Field>
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
