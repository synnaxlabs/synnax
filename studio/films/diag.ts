import { TimeRange, TimeSpan } from "@synnaxlabs/client";
import { capture, film, fixtures } from "@/index";
export const viewport = { width: 1440, height: 900 };
export default async (session: capture.CaptureSession): Promise<void> => {
  const errors: string[] = [];
  session.page.on("console", (m) => { if (m.type() === "error" || m.type() === "warning") errors.push(m.text().slice(0, 300)); });
  session.page.on("worker", (w) => errors.push(`worker ${w.url().slice(0, 120)}`));
  const world = await fixtures.testStand({ now: () => session.now() });
  try {
    await capture.login(session, world.project);
    await session.click(session.page.getByText("Primary", { exact: true }).first());
    await session.settle(1500);
    const { page } = session;
    await session.click(page.locator(".console-controls button:has(svg.pluto-icon--circle)").first());
    await session.settle(800);
    errors.push("--- acquired");
    await session.click(page.locator(".pluto-diagram").first().getByRole("button", { name: "Start Sequence" }).first());
    errors.push("--- started");
    await session.settle(4000);
    await page.screenshot({ path: "out/films/diag-after.png" });
    const c = fixtures.connect();
    const now = session.now();
    const fr = await c.read(new TimeRange(now.sub(TimeSpan.seconds(90)), now.add(TimeSpan.hours(1))), "ox_pt_1");
    console.log("virtual now", now.toString(), "samples", fr.length);
    const all = await c.read(new TimeRange(now.sub(TimeSpan.hours(2)), now.add(TimeSpan.hours(2))), "daq_time");
    const logs = await c.read(new TimeRange(now.sub(TimeSpan.hours(2)), now.add(TimeSpan.hours(2))), "sequence_log");
    console.log("logs", Array.from(logs));
    const cmd = await c.read(new TimeRange(now.sub(TimeSpan.hours(2)), now.add(TimeSpan.hours(2))), "ox_vent_cmd");
    console.log("ox_vent_cmd", Array.from(cmd));
    console.log("daq_time", all.length, all.length > 0 ? [String(all.at(0)), String(all.at(-1))] : []);
    await c.close();
    session.startRecording();
    await session.mark("a");
    await session.hold(100);
    await session.mark("b");
  } finally {
    console.log(errors.slice(0, 40).join("\n"));
    await world.stop();
  }
};
export const edit = film.edit([{ type: "card", lines: ["x"] }]);
