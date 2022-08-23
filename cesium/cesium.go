package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/cesium/internal/segment"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/signal"
)

var (
	// NotFound is returned when a channel or a range of data cannot be found in the DB.
	NotFound = query.NotFound
	// UniqueViolation is returned when a provided channel key already exists in the DB.
	UniqueViolation = query.UniqueViolation
)

type DB interface {

	// NewCreate opens a new Create query that is used for writing data to the DB.
	// A simple, synchronous create query looks is the following:
	//
	//      // Open a DB with a memory backed file system.
	//		db := cesium.Open("", cesium.MemBacked())
	//
	//      // Create a new channel that samples five float64 values per second. StorageKeys
	//      // will automatically generate a sequential uint16 key for the channel.
	//      // It is possible to specify a custom, UNIQUE key for the channel.
	//      key, err := cesium.CreateChannel(cesium.Channel{
	//          Density: cesium.Float64,
	//          Rate: 5 * cesium.Hz,
	//		})
	//		if err != nil {
	// 	    	 log.Fatal(err)
	//		}
	//
	//	    // Create a new segment to write. If you don't know what segments are,
	//	    // check out the cesium.Segment documentation.
	//      segments := []cesium.Segment{
	//    		ChannelKey: key,
	//          Start: cesium.Now(),
	//          data: cesium.MarshalFloat64([]{1.0, 2.0, 3.0})
	//		}
	//
	//		// db.Sync is a helper that turns a typically asynchronous write into an
	//		// acknowledged, synchronous write.
	//	    err := db.Sync(ctx, db.NewCreate().WhereChannels(key), &segments)
	//		if err != nil {
	//			logger.Fatal(err)
	//		}
	//
	// The above example will create a new channel with the type Float64 and a data rate of 5 Hz.
	// It will then write a segment with 3 samples to the database.
	//
	// The Create query acquires a write lock on the channels provided to the query
	// (in create.WhereChannels). No other goroutines can write to the locked channels
	// until the query completed.
	//
	// Asynchronous Create queries are default in cesium to allow for network
	// optimization and multi-segment write locks. They are a bit more complex to
	// write, however.  See the following example:
	//
	//		// Assuming DB is opened, channel is created, and a segment is defined.
	//	 	// See above example for details.
	//
	//	    // Open the create query for the channel. The first return value
	//     // is a channel where segments are written. The second is a response channel
	//      // containing any errors encountered during writes. The last value is an
	//      // error value that is returned if the query fails to open properly.
	//		// RouteStream for details on what each return value does.
	//		req, res, err := db.NewCreate().WhereChannels(key).RouteStream(ctx)
	//
	//		// Write the segment to the Create Request RouteStream.
	//      req <- cesium.CreateRequest{Segments: segments}
	//
	//		// Close the request stream. This lets the query know its safe to shut down
	// 		// operations and release locks.
	//		close(req)
	//
	//		// Wait for the Create query to acknowledge all writes. The Create query
	//	 	// will close the response channel when all segments are durable.
	// 		for resV := range res {
	//			if resV.err != nil {
	//				logger.Fatal(resV.err)
	//			}
	//		}
	//
	//		// Do what you want, but remember to close the database when done.
	//
	// Although waiting for the response channel to close is a common pattern for
	// Create queries, it is not required. StorageKeys will ensure all writes are
	// acknowledged upon DB.Close.
	//
	// It's important to note that the Create query does NOT operate in an atomic
	// manner. It is possible for certain segments to be persisted while others are not.
	NewCreate() Create

	// NewRetrieve opens a new Retrieve query that is used for retrieving data from the DB. A simple, synchronous
	// retrieve query looks like the following:
	//
	// 	 	// Open the DB, create a channel, and write some data to it. See NewCreate
	//	 	// for details on how to accomplish this.
	//
	//		// Open the Retrieve query, and read the results from disk into res.
	//		// DB.Sync is a helper that turns an asynchronous retrieve in a
	//		// synchronous one.
	//		var res []cesium.Segment
	// 		q := db.NewRewRetrieve().WhereChannels(key)
	//		if err := db.syncExec(ctx, q, &res); err != nil {
	//			log.Fatal(err)
	//		}
	//
	// The above example retrieves all data from the channel and binds it into res.
	// It's possible to retrieve a subset of data by time range by using the
	// Retrieve.WhereTimeRange method.
	//
	// Notes on Segmentation of data:
	//
	//		Retrieve results returned as segments (segment). The segments are not
	//	 	guaranteed to be in chronological order. This is a performance optimization
	//	 	to allow for more efficient data retrieval.
	//
	//	 	Segments are also not guaranteed to be contiguous. Because Create pattern
	//	 	cesium uses, it's possible to leave time gaps between segments
	//	 	(these represent times when that a particular sensor/sensor/emitter was
	//	 	inactive).
	//
	// Retrieve may return the following errors upon opening:
	//
	//		NotFound - The channel was not found in the database or the time range
	//		provided contained no data.
	//
	// Asynchronous Retrieve queries are the default in cesium. This allows for
	// network optimization (i.e. send the data across the network as you read more
	// data from IO). However, they are a more complex to write, however. See the
	// following example:
	//
	// 		// Assuming DB is opened and two segment have been created for a channel
	//	 	// with key 'key'. See NewCreate for details on how to accomplish this.
	//      // Start the retrieve query. The first return value is a channel that
	// 		// segments from disk are written to. The second is an error encountered
	// 		// during query open. To cancel a query abruptly, cancel the context
	// 		// provided to the Retrieve.Stream method.
	// 		ctx, cancel := context.WithCancel(context.Background())
	//		res, err := db.NewRetrieve().
	//						WhereTimeRange(cesium.TimeRangeMax).
	//						WhereChannels(key).
	//						RouteStream(ctx)
	//
	//      var res []cesium.Segment
	//
	//		// Retrieve will close the response channel when it has finished reading
	//      // all segments from disk.
	// 		for _, resV := range res {
	//			if resV.Error != nil {
	//				logger.Fatal(resV.Error)
	//			}
	//			res = append(res, res.Segments...)
	//		}
	//
	//      // do what you want with the data, just remember to close the database when done.
	//
	// It's also possible to iterate over the data in a set of channels. To do this,
	// call Retrieve.Iterate on the query instead of Retrieve.Stream:
	//
	//		ctx, cancel := context.WithCancel(context.Background())
	//		iter := db.NewRetrieve().
	//						WhereTimeRange(cesium.TimeRangeMax).
	//						WhereChannels(key).
	//						Write(ctx)
	//
	//      // It's important to check for errors before proceeding.
	//      if err := iter.Error(); err != nil {
	//         log.Fatal(err)
	//      }
	//
	//		// Start a goroutine that reads data returned from the iterator.
	//		go func() {
	//		for {
	//          // Open a stream with a buffer value of ten and pipe values from
	//          // the iterator to it.
	//			stream := confluence.NewStream[cesium.RetrieveResponse](10)
	//			iter.OutTo(stream)
	//
	//			for res := range stream.Output() {
	//				if res.Error != nil {
	//					log.Fatal(res.Error)
	//				}
	//				// Do something with the data.
	//				sendOverTheNetwork(res.Segments)
	// 			}
	//		}()
	//
	//      // Seek to the start of the iterator.
	//		iter.SeekFirst()
	//
	//      // Read the next 15 minutes of data. valid will be true if any data exists
	// 		// within the span. valid will return false if any errors are encountered.
	//      valid := iter.NextSpan(15 * time.Minutes)
	//
	//	    // Close the iterator. This will close any response channels bound by
	//	    // calling OutTo.
	//		if err := iter.Close(); err != nil {
	//			log.Fatal(err)
	//		}
	//
	NewRetrieve() Retrieve

	// CreateChannel opens a new CreateChannel query that is used for creating a
	// new channel in the DB. Creating a channel is simple:
	//
	//		// Open the DB
	//		ctx := context.Background()
	//		db := cesium.Open("", cesium.MemBacked())
	//
	//		// Create a channel. The generated key can be used to write data to and
	// 		// retrieve data from the channel.
	//		key, err := cesium.CreateChannel(cesium.Channel{
	//           Rate: 5 *cesium.Hz,
	//           Density: cesium.Float64,
	//		})
	//		if err != nil {
	//			logger.Fatal(err)
	//		}
	//		fmt.Println(key)
	//		// output:
	//		//  1
	//
	// If the cesium.channel.Field field is not set, the DB will automatically generate
	// an auto-incrementing uint16 key for you. StorageKeys requires that all channels have
	// a unique key. If you attempt to create a channel with a duplicate key, the DB
	// will return a UniqueViolation error.
	CreateChannel(ch Channel) (ChannelKey, error)

	// RetrieveChannel retrieves information about the channels with the specified keys
	// from the DB. Retrieving a channel is simple.
	//
	// 		// Assuming DB is opened and a channel with key 1 has been created.
	//		// See CreateChannel for details on how to accomplish this.
	//
	//		// Retrieve the channel.
	//		ch, err := cesium.RetrieveChannel()
	//		if err != nil {
	//			logger.Fatal(err)
	//		}
	//		fmt.Println(ch.Field)
	//		// output:
	//		//  1
	//
	// If any of the channels with the provided keys cannot be found, DB will return
	// a NotFound error.
	RetrieveChannel(keys ...ChannelKey) ([]Channel, error)

	// Sync is a utility that executes a query synchronously. It is useful for operations
	// that require all data to be persisted/returnee before continuing.
	//
	// Sync only supports Create and Retrieve queries, and will panic if any other
	// entity is passed. In the case of a Create query, the 'segments' arg represents
	// the data to write to the DB. A Retrieve query will do the reverse,
	// binding returned data into 'segments' instead.
	//
	// For examples on how to use Sync, see the documentation for NewCreate and NewRetrieve.
	Sync(ctx context.Context, query interface{}, segments *[]Segment) error

	// Close closes the DB. Close ensures that all queries are complete and all data
	//is persisted to disk. Close will block until all queries are completed,
	// so make sure to stop any running queries before calling.
	Close() error
}

type (
	Channel    = channel.Channel
	ChannelKey = channel.Key
	Segment    = segment.Segment
)

type db struct {
	kv                kvx.DB
	externalKV        bool
	wg                signal.WaitGroup
	shutdown          context.CancelFunc
	create            query.Factory[Create]
	retrieve          query.Factory[Retrieve]
	channelKeyCounter *kvx.PersistedCounter
}

// NewCreate implements DB.
func (d *db) NewCreate() Create { return d.create.New() }

// NewRetrieve implements DB.
func (d *db) NewRetrieve() Retrieve { return d.retrieve.New() }

// CreateChannel implements DB.
func (d *db) CreateChannel(ch Channel) (ChannelKey, error) {
	channelKV := kv.NewChannel(d.kv)
	if ch.Key != 0 {
		exists, err := channelKV.Exists(ch.Key)
		if err != nil {
			return 0, err
		}
		if exists {
			return 0, UniqueViolation
		}
	} else {
		key, err := d.channelKeyCounter.Add()
		if err != nil {
			return 0, err
		}
		ch.Key = ChannelKey(key)
	}
	return ch.Key, channelKV.Set(ch)
}

// RetrieveChannel implements DB.
func (d *db) RetrieveChannel(keys ...ChannelKey) ([]Channel, error) {
	return kv.NewChannel(d.kv).Get(keys...)
}

// Sync implements DB.
func (d *db) Sync(ctx context.Context, query interface{}, seg *[]Segment) error {
	return syncExec(ctx, query, seg)
}

// Close implements DB.
func (d *db) Close() error {
	d.shutdown()
	err := d.wg.Wait()
	if !d.externalKV {
		if kvErr := d.kv.Close(); kvErr != nil {
			return kvErr
		}
	}
	if err != context.Canceled {
		return err
	}
	return nil
}
