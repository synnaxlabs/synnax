#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from bench.config import (
    ITERATIONS,
    new_data_iterator,
)
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
        # delete the volume
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

def bench_timescale():
    with start_timescaledb():
        # Connect to TimescaleDB
        conn = psycopg2.connect(
            host=HOST,
            port=PORT,
            user=USER,
            password=PASSWORD,
            dbname=DB_NAME
        )
        cur = conn.cursor()

        # Create the table with 'time' and 'data' columns
        create_table_query = '''
        CREATE TABLE IF NOT EXISTS data (
            time TIMESTAMPTZ NOT NULL,
            data DOUBLE PRECISION
        );
        SELECT create_hypertable('data', 'time', if_not_exists => TRUE);
        '''
        cur.execute(create_table_query)
        conn.commit()

        total_time = sy.TimeSpan.SECOND * 0
        for i, df in enumerate(new_data_iterator()):
            perf_start = sy.TimeStamp.now()
            print("iter start")

            # Convert 'time' from int64 nanoseconds to datetime
            df['time'] = pd.to_datetime(df['time'], unit='ns')

            # Ensure 'data' is float64 for consistency
            df['data'] = df['data'].astype(float)

            # Convert DataFrame to list of tuples for insertion
            # records = list(df[['time', 'data']].itertuples(index=False, name=None))
            df = df[['time', 'data']]


            # Convert DataFrame to CSV format in memory
            output = io.StringIO()
            df.to_csv(
                output,
                sep='\t',
                header=False,
                index=False,
                date_format='%Y-%m-%d %H:%M:%S.%f%z'
            )
            output.seek(0)  # Move to the start of the StringIO object

            # Use COPY FROM STDIN to bulk load the data
            copy_query = "COPY data (time, data) FROM STDIN WITH (FORMAT csv, DELIMITER E'\\t')"
            cur.copy_expert(copy_query, output)
            conn.commit()

            total_time += sy.TimeStamp.since(perf_start)
            print(f"Iteration {i + 1}/{ITERATIONS} completed.")

        cur.close()
        conn.close()
        return total_time
