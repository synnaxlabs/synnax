import { combineReducers } from "@reduxjs/toolkit";
import {
  slice as driftSlice,
  TauriWindow,
  configureStore,
} from "@synnaxlabs/drift";
import clusterSlice from "./features/cluster/store/slice";
import mosaicSlice from "./mosaic/slice";

export default configureStore({
  window: new TauriWindow(),
  reducer: combineReducers({
    cluster: clusterSlice.reducer,
    drift: driftSlice.reducer,
    mosaic: mosaicSlice.reducer,
  }),
});
