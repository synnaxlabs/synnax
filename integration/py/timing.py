import synnax as sy


def time_write(file_name):
    def decorator(func):
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
            with open(file_name, "a") as f:
                f.write(s)

        return wrapper
    return decorator


def time_read(file_name):
    def decorator(func):
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
            with open(file_name, "a") as f:
                f.write(s)

        return wrapper
    return decorator


def time_delete(file_name):
    def decorator(func):
        def wrapper(*args):
            start = sy.TimeStamp.now()
            func(*args)
            end = sy.TimeStamp.now()

            time: sy.TimeSpan = start.span(end)
            params = args[0]
            s = f'''
-- Test Node Delete ({params.identifier})--
Time taken: {time}
Configuration:
    Number of channels: {len(params.channels)}
    Time Range: {params.time_range}
                '''
            with open(file_name, "a") as f:
                f.write(s)

        return wrapper
    return decorator


def time_stream(file_name):
    def decorator(func):
        def wrapper(*args):
            start = sy.TimeStamp.now()
            samples = func(*args)
            end = sy.TimeStamp.now()

            time: sy.TimeSpan = start.span(end)
            params = args[0]
            samples_per_second = samples / (float(time) / float(sy.TimeSpan.SECOND))
            s = f'''
-- Test Node Stream ({params.identifier})--
Samples streamed: {samples}
Time taken: {time}
Calculated Samples per Second: {samples_per_second:,.2f}
Configuration:
    Number of streamers: 1
    Number of channels: {len(params.channels)}
    Close after frames: {params.close_after_frames}

                '''
            with open(file_name, "a") as f:
                f.write(s)

        return wrapper
    return decorator

