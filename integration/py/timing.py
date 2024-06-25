import synnax as sy

FILE_NAME = "timing.log"

def time_write(func):
    def wrapper(*args):
        start = sy.TimeStamp.now()
        func(*args)
        end = sy.TimeStamp.now()

        time: sy.TimeSpan = start.span(end)
        params = args[0]
        samples = params.num_channels() * params.samples_per_domain * params.domains
        samples_per_second = samples / (float(time) / float(sy.TimeSpan.SECOND))
        s = f'''
-- Python Write ({params.identifier}) --
Samples written: {samples}
Time taken: {time}
Calculated Samples per Second: {samples_per_second:,.2f}
Configuration:
Number of writers: {params.num_writers}
Number of channels: {params.num_channels()}
Number of domains: {params.domains:,.0f}
Samples per domain: {params.samples_per_domain:,.0f}
Auto commit: {str(params.auto_commit)}
Index persist interval: {params.index_persist_interval}
Writer mode: {sy.WriterMode(params.writer_mode).name}

            '''
        with open(FILE_NAME, "a") as f:
            f.write(s)

    return wrapper


def time_read(func):
    def wrapper(*args):
        start = sy.TimeStamp.now()
        samples = func(*args)
        end = sy.TimeStamp.now()

        time: sy.TimeSpan = start.span(end)
        params = args[0]
        samples_per_second = samples / (float(time) / float(sy.TimeSpan.SECOND))
        s = f'''
-- Python Read ({params.identifier})--
Samples read: {samples}
Time taken: {time}
Calculated Samples per Second: {samples_per_second:,.2f}
Configuration:
Number of iterators: {params.num_iterators}
Number of channels: {params.num_channels()}
Chunk size: {params.chunk_size:,.0f}

            '''
        with open(FILE_NAME, "a") as f:
            f.write(s)

    return wrapper


def time_delete(func):
    def wrapper(*args):
        start = sy.TimeStamp.now()
        func(*args)
        end = sy.TimeStamp.now()

        time: sy.TimeSpan = start.span(end)
        params = args[0]
        s = f'''
-- Node Delete ({params.identifier})--
Time taken: {time}
Configuration:
Number of channels: {len(params.channels)}
Time Range: {params.time_range}
            '''
        with open(FILE_NAME, "a") as f:
            f.write(s)

    return wrapper


def time_stream(func):
    def wrapper(*args):
        start = sy.TimeStamp.now()
        samples = func(*args)
        end = sy.TimeStamp.now()

        time: sy.TimeSpan = start.span(end)
        params = args[0]
        samples_per_second = samples / (float(time) / float(sy.TimeSpan.SECOND))
        s = f'''
-- Python Stream ({params.identifier})--
Samples streamed: {samples}
Time taken: {time}
Calculated Samples per Second: {samples_per_second:,.2f}
Configuration:
Number of streamers: 1
Number of channels: {len(params.channels)}
Close after frames: {params.close_after_frames}

            '''
        with open(FILE_NAME, "a") as f:
            f.write(s)

    return wrapper

