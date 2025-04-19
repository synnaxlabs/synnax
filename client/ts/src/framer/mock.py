 

first_frame = streamer.read(0.01)
 with client.open_writer(
        channels=[WRITE_CHANNEL, WRITE_INDEX],
        start=first_frame[READ_INDEX][-1],
        enable_auto_commit=True,
    ) as writer:
        print(f"streaming {READ_CHANNEL}, index {READ_INDEX}")
        print(f"writing {WRITE_CHANNEL}, index {WRITE_INDEX}")
        data = {
            WRITE_CHANNEL: first_frame[READ_CHANNEL][-1],
            WRITE_INDEX: first_frame[READ_INDEX][-1],
        }
        writer.write(data)
        while True:
            frame = streamer.read(0.01)
            if frame is None:
                continue
            data = {
                WRITE_CHANNEL: frame[READ_CHANNEL][-1],
                WRITE_INDEX: frame[READ_INDEX][-1],
            }
            print(f"writing: ", data)
            if not writer.write(data):
                break
            
        writer.error()