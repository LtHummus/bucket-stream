package dynamicreader

import "io"

type DynamicReader struct {
	currentReader io.ReadCloser
	getNextReader func() (io.ReadCloser, error)
}

func NewDynamicReader(getNextReader func() (io.ReadCloser, error)) *DynamicReader {
	return &DynamicReader{
		getNextReader: getNextReader,
	}
}

func (dr *DynamicReader) Read(p []byte) (int, error) {
	for {
		if dr.currentReader == nil {
			next, err := dr.getNextReader()
			if err != nil {
				dr.currentReader.Close()
				return 0, err
			}

			// if there are no more readers to fetch, return EOF
			if next == nil {
				return 0, io.EOF
			}
			dr.currentReader = next
		}

		n, err := dr.currentReader.Read(p)
		if err != nil {
			dr.currentReader.Close()
			// stop using the current reader
			dr.currentReader = nil

			if err == io.EOF {
				// if we read some bytes, we still need to return them
				if n > 0 {
					return n, nil
				}

				// otherwise, loop again to get a new reader and try again
				continue
			}

			return n, err
		}

		return n, nil
	}
}
