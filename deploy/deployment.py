import docker
import requests
import time
import logging

# TODO: expose these as environment variables

image_name = "synnaxlabs/synnax"  # image name
image_tag = "latest"  # image tag
container_name = "synnax"  # container name
port = "9090"  # host port to expose
container_port = "9090"  # container port to expose
mount_host = "/home/masa/"  # host mount point
mount_container = "/var/"  # container mount point

retry_count = 10  # number of retries to check local container availability
retry_interval = 6  # seconds between retries

poll_interval = 300  # seconds between polls

args = "-vmi"  # container args

# Local container API endpoint to check availability
local_container_url = "https://www.google.com"  # "http://127.0.0.1:9090"  # TODO: Impl container update API
# currently using google.com as a dummy endpoint to trigger instant container update on image update

# Docker client
client = docker.from_env()

logging.basicConfig(level=logging.INFO)

if not client.containers.list(filters={"name": container_name}):
    logging.log(logging.INFO, "Local container not found. Creating.")
    client.containers.run(
        f"{image_name}:{image_tag}",
        name=container_name,
        detach=True,
        ports={port: container_port},
        restart_policy={"Name": "always"},
        volumes={mount_host: {"bind": mount_container, "mode": "rw"}},
        command=args,
    )
    logging.log(logging.INFO, "Local container created.")

local_image_id = client.containers.get(container_name).image.id

logging.log(logging.INFO, "Local image ID: {}".format(local_image_id))

while True:
    try:
        # Poll Docker Hub for the latest image
        hub_image = client.images.pull(f"{image_name}:{image_tag}")
        hub_image_id = hub_image.id
        # Check if a new image is available on Docker Hub
        if hub_image_id != local_image_id:
            logging.log(
                logging.INFO, "New image detected, image ID: {}".format(hub_image_id)
            )
            logging.log(logging.INFO, "Checking local container availability.")
            for i in range(retry_count):
                try:
                    logging.log(
                        logging.INFO, "Attempt #{} of {}.".format(i + 1, retry_count)
                    )
                    response = requests.get(local_container_url)
                    if response.status_code == 200:
                        # if response.json()['status'] == 'ok': # TODO: Impl container update API
                        logging.log(
                            logging.INFO, "Local container is available for update."
                        )
                        # Perform the update
                        client.containers.get(container_name).stop()
                        logging.log(logging.INFO, "Container stopped.")
                        client.containers.get(container_name).remove()
                        logging.log(logging.INFO, "Container removed.")
                        client.containers.run(
                            f"{image_name}:{image_tag}",
                            name=container_name,
                            detach=True,
                            ports={port: container_port},
                            restart_policy={"Name": "always"},
                            volumes={
                                mount_host: {"bind": mount_container, "mode": "rw"}
                            },
                            command=args,
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

                time.sleep(retry_interval)
        local_image_id = hub_image_id
    except docker.errors.APIError:
        logging.log(logging.ERROR, "Error polling Docker Hub.")
    except KeyboardInterrupt:
        logging.log(logging.INFO, "Exiting.")
        break

    try:
        time.sleep(poll_interval)
    except KeyboardInterrupt:
        logging.log(logging.INFO, "Keyboard interrupt. Exiting.")
        break
