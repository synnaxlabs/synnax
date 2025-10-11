#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import docker
import requests
import time
import logging

# TODO: expose these as environment variables

IMAGE_NAME = "synnaxlabs/synnax"  # image name
IMAGE_TAG = "latest"  # image tag
CONTAINER_NAME = "synnax"  # container name
HOST_PORT = "9090"  # host port to expose
CONTAINER_PORT = "9090"  # container port to expose
HOST_MOUNT = "/home/masa/"  # host mount point
CONTAINER_MOUNT = "/var/"  # container mount point

RETRY_COUNT = 10  # number of retries to check local container availability
RETRY_INTERVAL = 6  # seconds between retries

POLL_INTERVAL = 300  # seconds between polls

ARGS = "-vmi"  # container args

# Local container API endpoint to check availability
LOCAL_CONTAINER_URL = "https://www.google.com"  # "http://127.0.0.1:9090"  # TODO: Impl container update API
# currently using google.com as a dummy endpoint to trigger instant container update on image update

# Docker client
client = docker.from_env()

logging.basicConfig(level=logging.INFO)

if not client.containers.list(filters={"name": CONTAINER_NAME}):
    logging.log(logging.INFO, "Local container not found. Creating.")
    client.containers.run(
        f"{IMAGE_NAME}:{IMAGE_TAG}",
        name=CONTAINER_NAME,
        detach=True,
        ports={HOST_PORT: CONTAINER_PORT},
        restart_policy={"Name": "always"},
        volumes={HOST_MOUNT: {"bind": CONTAINER_MOUNT, "mode": "rw"}},
        command=ARGS,
    )
    logging.log(logging.INFO, "Local container created.")

local_image_id = client.containers.get(CONTAINER_NAME).image.id

logging.log(logging.INFO, "Local image ID: {}".format(local_image_id))

while True:
    try:
        # Poll Docker Hub for the latest image
        hub_image = client.images.pull(f"{IMAGE_NAME}:{IMAGE_TAG}")
        hub_image_id = hub_image.id
        # Check if a new image is available on Docker Hub
        if hub_image_id != local_image_id:
            logging.log(
                logging.INFO, "New image detected, image ID: {}".format(hub_image_id)
            )
            logging.log(logging.INFO, "Checking local container availability.")
            for i in range(RETRY_COUNT):
                try:
                    logging.log(
                        logging.INFO, "Attempt #{} of {}.".format(i + 1, RETRY_COUNT)
                    )
                    response = requests.get(LOCAL_CONTAINER_URL)
                    if response.status_code == 200:
                        # if response.json()['status'] == 'ok': # TODO: Impl container update API
                        logging.log(
                            logging.INFO, "Local container is available for update."
                        )
                        # Perform the update
                        client.containers.get(CONTAINER_NAME).stop()
                        logging.log(logging.INFO, "Container stopped.")
                        client.containers.get(CONTAINER_NAME).remove()
                        logging.log(logging.INFO, "Container removed.")
                        client.containers.run(
                            f"{IMAGE_NAME}:{IMAGE_TAG}",
                            name=CONTAINER_NAME,
                            detach=True,
                            ports={HOST_PORT: CONTAINER_PORT},
                            restart_policy={"Name": "always"},
                            volumes={
                                HOST_MOUNT: {"bind": CONTAINER_MOUNT, "mode": "rw"}
                            },
                            command=ARGS,
                        )
                        logging.log(logging.INFO, "Container updated.")
                        break
                    else:
                        logging.log(
                            logging.WARN, "Local container is not available for update."
                        )
                except requests.exceptions.ConnectionError:
                    logging.log(
                        logging.ERROR,
                        "Error connecting to local container to check availability.",
                    )

                time.sleep(RETRY_INTERVAL)
        local_image_id = hub_image_id
    except docker.errors.APIError:
        logging.log(logging.ERROR, "Error polling Docker Hub.")
    except KeyboardInterrupt:
        logging.log(logging.INFO, "Exiting.")
        break

    try:
        time.sleep(POLL_INTERVAL)
    except KeyboardInterrupt:
        logging.log(logging.INFO, "Keyboard interrupt. Exiting.")
        break
