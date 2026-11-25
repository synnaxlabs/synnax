# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Creates the sample project and the preview workspace on the Core at the URL it takes.
# Build it from the repository root after the bundle in console/scripts/vite.config.ts.

FROM node:26.13.1-slim@sha256:abe7e5a82b61a8ef67b315b85497c9e65173387ccf2f4a029248184b8e020be5

WORKDIR /app
COPY console/scripts/dist dist
COPY console/src/feature/project/*.arc src/feature/project/
USER node

ENTRYPOINT ["/bin/sh", "-c", \
    "node dist/create_sample.js \"$0\" && node dist/create_preview_workspace.js \"$0\""]
