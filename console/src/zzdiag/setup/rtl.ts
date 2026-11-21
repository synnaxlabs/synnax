const { configure } = await import("@testing-library/react");
configure({ asyncUtilTimeout: 5000 });
const { preloadTimeLanguage } = await import("@synnaxlabs/lyra/testutil");
preloadTimeLanguage();
