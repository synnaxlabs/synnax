import madge from "madge";

madge("./src", {
  fileExtensions: ["ts", "tsx", "js", "jsx"],
  tsConfig: "tsconfig.json",
  detectiveOptions: {
    ts: {
      skipTypeImports: true,
    },
    tsx: {
      skipTypeImports: true,
    },
  },
})
  .then((res) => {
    console.log("Warnings:", res.warnings());
  })
  .catch((err) => {
    console.error(err);
  });
