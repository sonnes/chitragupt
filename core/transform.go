package core

// Transformer mutates a Transcript in place.
type Transformer interface {
	Transform(t *Transcript) error
}

// Chain applies transformers in order, stopping at the first error.
func Chain(t *Transcript, transformers ...Transformer) error {
	for _, tr := range transformers {
		if err := tr.Transform(t); err != nil {
			return err
		}
	}
	return nil
}
