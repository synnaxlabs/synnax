import { combineReducers } from "@reduxjs/toolkit";
import { slice, TauriWindow, configureStore } from "@synnaxlabs/drift";

export default configureStore({
  window: new TauriWindow(),
  reducer: combineReducers({ drift: slice.reducer }),
});
