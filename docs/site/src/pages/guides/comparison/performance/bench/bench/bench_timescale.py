from synnax import DataType

from bench.config import TestConfig
from bench.dockerutil import run_container, peak_stats
import io
import synnax as sy
from contextlib import contextmanager
import docker
import time
import psycopg2
import pandas as pd
import psycopg2.extras

PORT = 5432
HOST = 'localhost'
USER = 'postgres'
PASSWORD = 'password'
DB_NAME = 'postgres'

@contextmanager
def start_timescaledb():
    client = docker.from_env()
    NAME = 'timescaledb_bench'
    VOLUME_NAME = 'timescaledb_data'

    # Create or get the Docker volume
    try:
        volume = client.volumes.get(VOLUME_NAME)
        print(f"Docker volume '{VOLUME_NAME}' already exists.")
        volume.remove()
    except docker.errors.NotFound:
        volume = client.volumes.create(VOLUME_NAME)
        print(f"Docker volume '{VOLUME_NAME}' created.")

    # Define volume bindings
    volumes = {
        VOLUME_NAME: {'bind': '/var/lib/postgresql/data', 'mode': 'rw'}
    }
    client.containers.run(
        "timescale/timescaledb-ha:pg16",
        detach=True,
        name=NAME,
        ports={'5432/tcp': PORT},
        environment={
            'POSTGRES_PASSWORD': PASSWORD,
            'POSTGRES_USER': USER,
            'POSTGRES_DB': DB_NAME,
        },
        volumes=volumes,
    )
    time.sleep(5)  # Allow the container time to initialize
    try:
        yield
    finally:
        c = client.containers.get(NAME)
        c.stop()
        c.remove()

IMAGE_NAME = "timescale/timescaledb-ha:pg16"
CONTAINER_NAME = "timescaledb_bench"
VOLUME_NAME = "timescaledb_data"
VOLUME_PATH = "/var/lib/postgresql/data"
PORTS = {"5432/tcp": 5432}
ENVIRONMENT = {
    'POSTGRES_PASSWORD': PASSWORD,
    'POSTGRES_USER': USER,
    'POSTGRES_DB': DB_NAME,
}

def health_check():
    try:
        conn = psycopg2.connect(
            host=HOST,
            port=PORT,
            user=USER,
            password=PASSWORD,
            dbname=DB_NAME
        )
        cur = conn.cursor()
        cur.execute("SELECT 1")
        cur.close()
        conn.close()
        return True
    except Exception:
        return False

def bench_timescale(cfg: TestConfig):
    with run_container(
        image=IMAGE_NAME,
        name=CONTAINER_NAME,
        volume_name=VOLUME_NAME,
        ports=PORTS,
        environment=ENVIRONMENT,
        volume_dir=VOLUME_PATH,
        health_check=health_check,
    ) as get_stats:
        # Connect to TimescaleDB
        conn = psycopg2.connect(
            host=HOST,
            port=PORT,
            user=USER,
            password=PASSWORD,
            dbname=DB_NAME
        )
        cur = conn.cursor()

        # Create the main table with 'timestamps' column
        create_table_query = '''
        CREATE TABLE IF NOT EXISTS data (
            timestamps TIMESTAMPTZ NOT NULL
        );
        '''
        cur.execute(create_table_query)
        conn.commit()

        # Add columns for each channel
        for ch in cfg.channels[1:]:  # Skipping the time channel
            data_type = ch.data_type
            column_name = ch.name

            if data_type in [DataType.FLOAT32, DataType.FLOAT64]:
                sql_data_type = 'DOUBLE PRECISION'
            elif data_type in [DataType.INT32, DataType.INT64]:
                sql_data_type = 'BIGINT'
            elif data_type == DataType.TIMESTAMP:
                sql_data_type = 'TIMESTAMPTZ'
            else:
                raise ValueError(f"Unsupported data type: {data_type}")

            # Add column if it doesn't exist
            alter_table_query = f'''
            ALTER TABLE data
            ADD COLUMN IF NOT EXISTS {column_name} {sql_data_type};
            '''
            cur.execute(alter_table_query)
            conn.commit()

        # Create hypertable
        create_hypertable_query = '''
        SELECT create_hypertable('data', 'timestamps', if_not_exists => TRUE);
        '''
        cur.execute(create_hypertable_query)
        conn.commit()

        total_time = sy.TimeSpan.SECOND * 0
        stats = list()
        for i, df in enumerate(cfg.frames(index=True)):


            # Reset index to convert 'timestamps' from index to column
            df.reset_index(inplace=True)

            # Convert 'timestamps' from int64 nanoseconds to datetime
            df['timestamps'] = pd.to_datetime(df['timestamps'], unit='ns')

            # Ensure correct data types for each channel
            for ch in cfg.channels[1:]:
                column_name = ch.name
                data_type = ch.data_type
                if data_type in [DataType.FLOAT32, DataType.FLOAT64]:
                    df[column_name] = df[column_name].astype(float)
                elif data_type in [DataType.INT32, DataType.INT64]:
                    df[column_name] = df[column_name].astype(int)
                elif data_type == DataType.TIMESTAMP:
                    df[column_name] = pd.to_datetime(df[column_name], unit='ns')
            output = io.StringIO()
            df.to_csv(
                output,
                sep='\t',
                header=False,
                index=False,
                date_format='%Y-%m-%d %H:%M:%S.%f%z'
            )
            output.seek(0)  # Move to the start of the StringIO object
            columns_list = ', '.join(df.columns)
            perf_start = sy.TimeStamp.now()
            copy_query = f"COPY data ({columns_list}) FROM STDIN WITH (FORMAT csv, DELIMITER E'\\t')"
            cur.copy_expert(copy_query, output)
            conn.commit()
            total_time += sy.TimeSpan.since(perf_start)
            stats.append(get_stats())
            print(f"Iteration {i + 1}/{cfg.iterations} completed.")
        cur.close()
        conn.close()
        return peak_stats(stats), total_time
