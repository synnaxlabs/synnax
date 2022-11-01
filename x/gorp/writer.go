package gorp

import "github.com/synnaxlabs/x/kv"

type KVWriter[K Key, E Entry[K]] struct {
	kv.Batch
	options
}

func WrapKVBatch[K Key, E Entry[K]](batch kv.Batch, opts ...Option) *KVWriter[K, E] {
	return &KVWriter[K, E]{Batch: batch, options: newOptions(opts...)}
}

func (w *KVWriter[K, E]) Write(entry E) error {
	prefix := typePrefix[K, E](w.options)
	data, err := w.encoder.Encode(entry)
	if err != nil {
		return err
	}
	key, err := w.encoder.Encode(entry.GorpKey())
	if err != nil {
		return err
	}
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	if err = w.Set(append(prefix, key...), data, entry.SetOptions()...); err != nil {
		return err
	}
	return nil
}

func (w *KVWriter[K, E]) WriteMany(entries []E) error {
	for _, entry := range entries {
		if err := w.Write(entry); err != nil {
			return err
		}
	}
	return nil
}

func (w *KVWriter[K, E]) Delete(key K) error {
	prefix := typePrefix[K, E](w.options)
	data, err := w.encoder.Encode(key)
	if err != nil {
		return err
	}
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	if err = w.Batch.Delete(append(prefix, data...)); err != nil {
		return err
	}
	return nil
}
