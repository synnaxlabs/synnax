import synnax as sy
import multiprocessing
from dataclasses import dataclass
import atexit
import warnings


SYNNAX_CONNECTION_PARAMETERS = {
    "host": "192.168.0.10",
    "port": 9090,
    "username": "synnax",
    "password": "seldon",
    "secure": False,
}

POSITION_ORIENTATION_PAIRS: list[tuple[str, str]] = [
    ("Boost 1", "Photodiode 1"),
    ("Boost 1", "Photodiode 2"),
    ("Boost 2", "Photodiode 1"),
    ("Boost 2", "Photodiode 2"),
    ("Boost 2", "Pressure"),
    ("Boost 3", "Photodiode 1"),
    ("Boost 3", "Photodiode 2"),
    ("Breech", "Photodiode 1"),
    ("Breech", "Shop Side"),
    ("Breech", "Street Side"),
    ("Breech", "Z"),
    ("Muzzle", "Photodiode 1"),
    ("Muzzle", "Photodiode 2"),
    ("Z Camera", "Induction 1"),
    ("Z Camera", "Induction 2"),
    ("Kick", "Kick"),
    ("Kick", "Temperature"),
]


def generate_channel_key(
    position: str, orientation: str, high_speed: bool = False
) -> str:
    if not high_speed:
        return f"{position}----{orientation}"
    return f"high-speed-{generate_channel_key(position, orientation)}"


pool_running = multiprocessing.Value("b", False)
queue = multiprocessing.Queue()
worker_process = None


@dataclass
class Message:
    position: str
    orientation: str
    high_speed: bool
    timestamps: list[int]
    values: list[float]


def write_to_synnax(
    position: str,
    orientation: str,
    high_speed: bool,
    timestamps: list[int],
    values: list[float],
):
    msg = Message(
        position=position,
        orientation=orientation,
        high_speed=high_speed,
        timestamps=timestamps,
        values=values,
    )
    queue.put(msg)


def synnax_worker(queue):
    print("Synnax Worker Started")
    pool = None
    try:
        pool = SynnaxPool()
    except Exception as e:
        warnings.warn(str(e))

    while True:
        message = queue.get()
        if message == "STOP":
            if pool is not None:
                pool.stop()
            return
        if pool is not None:
            pool.write(
                position=message.position,
                orientation=message.orientation,
                high_speed=message.high_speed,
                timestamps=message.timestamps,
                values=message.values,
            )


def start_synnax_worker():
    global worker_process
    with pool_running.get_lock():
        if not pool_running.value:
            worker_process = multiprocessing.Process(
                target=synnax_worker, args=(queue,)
            )
            worker_process.start()
            pool_running.value = True


@atexit.register
def cleanup():
    print("Shutting Down Synnax Worker")
    global worker_process
    if worker_process and worker_process.is_alive():
        queue.put("STOP")
        worker_process.join()
    print("Synnax Worker Shut Down")


class SynnaxPool:
    client: sy.Synnax
    low_speed_writer: sy.Writer
    pairs_map: dict[str, tuple[int, int]]

    def __init__(self):
        self.writers = dict()
        self.client = sy.Synnax(**SYNNAX_CONNECTION_PARAMETERS)
        low_speed = list()
        self.pairs_map = dict()
        for position, orientation in POSITION_ORIENTATION_PAIRS:
            low_speed.extend(self.process_channel(position, orientation, False))
            self.process_channel(position, orientation, True)
        self.low_speed_writer = self.client.open_writer(
            start=sy.TimeStamp.now(),
            enable_auto_commit=True,
            channels=low_speed,
        )

    def stop(self):
        self.low_speed_writer.close()
        self.client.close()

    def process_channel(
        self, position: str, orientation: str, high_speed: bool = False
    ) -> tuple[int, int]:
        id_ = generate_channel_key(position, orientation, high_speed)
        speed_prefix = "High Speed" if high_speed else "Low Speed"

        name_prefix = f"{position} {orientation} {speed_prefix}"
        new_time_key = self.create_or_retrieve_channel(
            name=f"{name_prefix} Time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        new_data_key = self.create_or_retrieve_channel(
            name=f"{name_prefix} Data",
            index=new_time_key,
        )
        self.pairs_map[id_] = (new_time_key, new_data_key)
        return new_time_key, new_data_key

    def create_or_retrieve_channel(
        self,
        name: str,
        data_type: sy.DataType = sy.DataType.FLOAT64,
        is_index: bool = False,
        index: int = 0,
    ) -> int:
        try:
            return self.client.channels.retrieve(name).key
        except sy.NotFoundError:
            return self.client.channels.create(
                name=name,
                data_type=data_type,
                is_index=is_index,
                index=index,
            ).key

    def write_low_speed(
        self, position: str, orientation: str, timestamps: list[int], values: list[float]
    ) -> tuple[sy.Writer, int, int]:
        channel_id = generate_channel_key(position, orientation, False)
        time_key, data_key = self.pairs_map[channel_id]
        self.low_speed_writer.write({time_key: timestamps, data_key: values})
        try:
            if not self.low_speed_writer.commit():
                self.low_speed_writer.close()
        except Exception as e:
            warnings.warn(str(e))

    def write(
        self,
        position: str,
        orientation: str,
        high_speed: bool,
        timestamps: list[int],
        values: list[float],
    ):
        if not high_speed:
            return self.write_low_speed(position, orientation, timestamps, values)
        channel_id = generate_channel_key(position, orientation, high_speed)
        time_key, data_key = self.pairs_map[channel_id]
        try: 
            self.client.write(timestamps[0], {
                time_key: timestamps,
                data_key: values
            })
        except Exception as e:
            warnings.warn(str(e))