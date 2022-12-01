const fs = await import("fs");

const packageVersion = "0.1.0";
const {
	repo: { repo, owner },
} = { repo: { repo: "synnax", owner: "synnaxlabs" } };

const baseURL = `https://github.com/${owner}/${repo}/releases/download/app-v${packageVersion}/`;

const darwinURL = baseURL + "Synnax.app.tar.gz";
const linuxURL = baseURL + `synnax_${packageVersion}_amd64.AppImage.tar.gz`;
const windowsURL = baseURL + `Synnax_${packageVersion}_x64_en-US.msi.zip`;
const pub_date = new Date().toISOString().replace(/\.\d{3}Z$/, "Z");

const data = {
	version: `v${packageVersion}`,
	notes: "Synnax Update",
	pub_date,
	platforms: {
		"darwin-x86_64": {
			signature: await (await fetch(darwinURL + ".sig")).text(),
			url: darwinURL,
		},
		"linux-x86_64": {
			signature: await (await fetch(linuxURL + ".sig")).text(),
			url: linuxURL,
		},
		"windows-x86_64": {
			signature: await (await fetch(windowsURL + ".sig")).text(),
			url: windowsURL,
		},
	},
};

fs.writeFileSync("delta/release-spec.json", JSON.stringify(data, null, 2));
