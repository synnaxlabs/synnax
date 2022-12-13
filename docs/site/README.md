<br />
<p align="center">
    <a href="https://docs.synnaxlabs.com">
        <img src="../media/logo/title-white-on-black.svg" width="45%"/>
    </a>
</p>

# Synnax Documentation Site

This directory contains the source for
the [Synnax documentation site](https://docs.synnaxlabs.com), which serves as
the primary knowledge base for the Synnax platform.

### Local Development

```bash
earthly +dev
```

This command starts a local development server and opens up a browser window.
Most changes are reflected live without having to restart the server. If this
command fails, move into the root directory of the `synnax` repository and run:

```bash
earthly +clean
```

This will remove all build artifacts. Then, run the `+dev` command again.

### Build

The site can be built using Earthly:

```
earthly +build
```

The built site will be output to `./dist`. To serve the site locally, run:

```
yarn serve
```

### Deployment

The documentation site is automatically deployed to GitHub Pages whenever a pull
request is merged into the `main` branch. There's no need to worry about manual
deployment!
