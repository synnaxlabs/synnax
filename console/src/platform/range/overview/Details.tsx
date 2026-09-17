// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/range/overview/Details.css";

import { ranger } from "@synnaxlabs/client";
import {
  Access,
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

import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Framer } from "@/platform/framer";
import { Label } from "@/platform/label";
import { Panel } from "@/platform/panel";
import { FavoriteButton } from "@/platform/range/FavoriteButton";

interface ParentRangeButtonProps {
  rangeKey: string;
}

const Internal = ({ rangeKey }: ParentRangeButtonProps): ReactElement | null => {
  const parent = Ranger.useParent({ id: ranger.ontologyID(rangeKey) });
  const openTab = Panel.useOpenTab();
  if (parent == null) return null;
  const Icon = Ranger.STAGE_ICONS[Ranger.getStage(parent.timeRange)];
  return (
    <Flex.Box x gap="small" align="center">
      <Text.Text weight={450} color={9}>
        Child range of
      </Text.Text>
      <Button.Button
        color={9}
        variant="text"
        gap="small"
        className={CSS.BE("range-overview", "parent-button")}
        onClick={() =>
          openTab({ variant: "resource", resource: ranger.ontologyID(parent.key) })
        }
      >
        <Icon />
        {parent.name}
      </Button.Button>
    </Flex.Box>
  );
};

const ParentRangeButton = (props: ParentRangeButtonProps): ReactElement => (
  <Errors.SuspenseBoundary loading={null}>
    <Internal {...props} />
  </Errors.SuspenseBoundary>
);

export interface DetailsProps {
  rangeKey: string;
}

export const Details: FC<DetailsProps> = ({ rangeKey }) => {
  const range = Ranger.use({ key: rangeKey });
  const now = TimeStamp.now().nanoseconds;
  // The form saves on every edit, so a subject who cannot write the range gets it
  // read-only rather than a field that reverts once the save is refused.
  const canEdit = Access.useUpdateGranted(ranger.ontologyID(rangeKey));
  const { form, status } = Ranger.useForm({
    query: { key: rangeKey },
    initialValues: {
      key: rangeKey,
      name: "",
      timeRange: { start: now, end: now },
      labels: [],
    },
    autoSave: true,
    mode: canEdit ? "normal" : "preview",
  });

  const handleLink = Core.useCopyLinkToClipboard();
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

  const promptDownloadCSVModal = Framer.useDownloadCSVModal();

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
          <Flex.Box x className={CSS.BE("range-overview", "actions")} gap="small">
            <Button.Copy
              text={getPythonCode}
              tooltip={`Copy Python code for ${name}`}
              tooltipLocation="bottom"
              variant="text"
              successMessage={`Copied Python code for ${name} to clipboard`}
              textColor={9}
            >
              <Icon.Python />
            </Button.Copy>
            <Button.Copy
              variant="text"
              text={getTypeScriptCode}
              tooltip={`Copy TypeScript code for ${name}`}
              tooltipLocation="bottom"
              successMessage={`Copied TypeScript code for ${name} to clipboard`}
              textColor={9}
            >
              <Icon.TypeScript />
            </Button.Copy>
            <Divider.Divider y />
            <Button.Button
              tooltip={`Download data for ${name} as CSV`}
              tooltipLocation="bottom"
              variant="text"
              textColor={9}
              onClick={() =>
                handleError(async () => {
                  await promptDownloadCSVModal({
                    timeRange: form.get<NumericTimeRange>("timeRange").value,
                    name,
                    channels: [],
                    icon: <Icon.Range />,
                  });
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
            <FavoriteButton range={range} size="medium" />
          </Flex.Box>
        </Flex.Box>
        <Flex.Box className={CSS.B("time-range")} x gap="medium" align="center">
          <Form.Field<number> path="timeRange.start" padHelpText={false} label="From">
            {(p) => (
              <Input.DateTime level="h4" variant="text" onlyChangeOnBlur {...p} />
            )}
          </Form.Field>
          <Icon.Arrow.Right
            className={CSS.BE("range-overview", "arrow-icon")}
            color={9}
          />
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
            {(p) => (
              <Label.SelectMultiple
                zIndex={100}
                variant="floating"
                className={CSS.BE("range-overview", "labels-select")}
                {...p}
              />
            )}
          </Form.Field>
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
