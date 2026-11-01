// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const STATUS = "Engine 1";

/**
 * Docs `device-drivers/pagerduty/alert-task/create`: run "Create PagerDuty alert
 * task", enter a routing key, add an alert on the Engine 1 status, treat errors as
 * critical, and fill the incident fields. The task is never started.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.createStatuses([{ key: "engine-1", name: STATUS }]);
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1800);

  await capture.commandPalette(session, "Create PagerDuty alert task");
  const form = page.locator(".console-task-configure").first();
  await session.waitFor(form);
  await session.hold(800);
  // The form fills most of the panel, so zoom on a point at its center.
  await capture.zoomCenter(session, form, 1.4);

  // The key typed is the field's own placeholder, which is never sent anywhere.
  const routingKey = form.locator("input[type=password]").first();
  const key = await routingKey.getAttribute("placeholder");
  if (key == null) throw new Error("The routing key field has no placeholder");
  await session.click(routingKey);
  session.setSpeed(3);
  await session.type(key, 60);
  session.setSpeed(1);
  await session.hold(500);

  await session.click(form.getByRole("button", { name: "Add alert" }).first());
  const trigger = form.getByText("Select status", { exact: true }).first();
  await session.waitFor(trigger);
  await session.hold(600);

  await session.click(trigger, { text: true });
  const search = page.locator(".lyra-select__dialog:visible input").first();
  await session.waitFor(search);
  await session.hold(400);
  await session.type(STATUS);
  const option = page
    .locator(".lyra-select__dialog:visible .lyra-list__item")
    .filter({ hasText: STATUS })
    .first();
  await session.waitFor(option);
  await session.hold(400);
  await session.click(option.getByText(STATUS, { exact: true }).first(), {
    text: true,
  });
  await session.hold(600);

  const critical = form
    .locator(".lyra-input__item")
    .filter({ has: page.getByText("Treat error as critical", { exact: true }) })
    .getByRole("checkbox")
    .first();
  await session.click(critical);
  await session.hold(600);

  // Each incident field's placeholder is the value the docs enter.
  for (const value of ["engine-1", "engines", "engine-failure"]) {
    const field = form.getByPlaceholder(value, { exact: true }).first();
    // The lower fields start below the visible part of the form.
    await field.scrollIntoViewIfNeeded();
    await session.click(field);
    await session.type(value, 80);
    await session.hold(300);
  }

  session.endZoom();
  // Clear of the form, so the filled fields read.
  await session.moveTo({ x: 756, y: 860 });
  await session.hold(2500);
};
