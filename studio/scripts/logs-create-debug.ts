import { capture } from "@/index";

/** Diagnostic: native palette-button click, rig typing, then native item click. */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });
  await capture.clearPanel(session);

  session.startRecording();
  const btn = page.locator(".console-palette button").first();
  await session.moveTo(btn);
  await btn.click();
  const input = page.locator(".console-palette__input input[role='textbox']");
  await session.waitFor(input);
  await session.hold(300);
  await session.type(">");
  await session.type("Create log");
  await session.hold(600);
  const items = await page.locator(".pluto-list__item").allTextContents();
  console.log("items:", JSON.stringify(items.slice(0, 3)));
  const item = page
    .locator(".pluto-list__item")
    .filter({ has: page.getByText("Create log", { exact: true }) })
    .first();
  await session.moveTo(item, { text: true });
  await item.click();
  await session.settle(2500);
  console.log("log count:", await page.locator(".pluto-log").count());
};
