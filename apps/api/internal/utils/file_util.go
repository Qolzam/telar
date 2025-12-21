package utils

import (
	"os"
	"sync"
)

func GetFilesContents(files ...string) (map[string][]byte, error) {
	var wg sync.WaitGroup
	var m sync.Mutex
	var firstErr error

	filesLength := len(files)
	contents := make(map[string][]byte, filesLength)
	wg.Add(filesLength)

	for _, file := range files {
		go func(file string) {
			defer wg.Done()
			content, err := os.ReadFile(file)

			if err != nil {
				m.Lock()
				if firstErr == nil {
					firstErr = err
				}
				m.Unlock()
				return
			}

			m.Lock()
			contents[file] = content
			m.Unlock()
		}(file)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return contents, nil
}
