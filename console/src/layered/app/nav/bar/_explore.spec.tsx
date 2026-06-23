import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { screen } from "@testing-library/react";
import { OS } from "@synnaxlabs/pluto";
import { describe, it } from "vitest";

import { Arc } from "@/arc";
import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { AuxTop, Left, Top } from "@/layered/app/nav/bar";
import { Session } from "@/layered/session";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Project } from "@/project";
import { Range } from "@/range";
import { Schematic } from "@/schematic";
import { Status } from "@/status";
import { Table } from "@/table";
import { renderWithConsole } from "@/testUtils";

const reducer = combineReducers({
  [Cluster.SLICE_NAME]: Cluster.reducer,
  [Docs.SLICE_NAME]: Docs.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [LinePlot.SLICE_NAME]: LinePlot.reducer,
  [Log.SLICE_NAME]: Log.reducer,
  [Session.Nav.SLICE_NAME]: Session.Nav.reducer,
  [Range.SLICE_NAME]: Range.reducer,
  [Schematic.SLICE_NAME]: Schematic.reducer,
  [Status.SLICE_NAME]: Status.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Project.SLICE_NAME]: Project.reducer,
  [Arc.SLICE_NAME]: Arc.reducer,
});

const store = () => configureStore({ reducer });

describe("explore", () => {
  it("detected OS", () => {
    console.log("OS =", OS.use({ default: "Linux" }));
  });
  it("Left", () => {
    renderWithConsole(<Left />, { store: store() });
    console.log("LEFT HTML:\n", document.body.innerHTML.slice(0, 2000));
  });
  it("Top", () => {
    renderWithConsole(<Top />, { store: store() });
    console.log("TOP TEXT:\n", screen.getByRole ? document.body.textContent : "");
  });
  it("AuxTop", () => {
    renderWithConsole(<AuxTop />, { store: store() });
    console.log("AUXTOP TEXT:\n", document.body.textContent);
  });
});
