package watcher

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type WatchResult struct {
	Old map[string][]byte
	New map[string][]byte
}

type Session struct {
	C      <-chan *WatchResult
	hashes map[string][]byte
	lock   *sync.Mutex
	exit   chan interface{}
	files  chan []string
	closed bool
}

func filterNullAndDirsAndDuplicatesFiles(files []string) []string {
	uniquePaths := make(map[string]bool)
	var uniqueFiles []string
	for _, file := range files {
		info, err := os.Stat(file)
		if os.IsNotExist(err) || info.IsDir() {
			continue
		}
		fileAbsolutePath, _ := filepath.Abs(file)
		if _, value := uniquePaths[fileAbsolutePath]; !value {
			uniquePaths[fileAbsolutePath] = true
			uniqueFiles = append(uniqueFiles, fileAbsolutePath)
		}
	}
	return uniqueFiles
}

func (s *Session) checkFiles(files []string) (bool, *WatchResult) {
	s.lock.Lock()
	defer s.lock.Unlock()
	oldHashes := s.hashes
	notChanged := true
	uniqueFiles := filterNullAndDirsAndDuplicatesFiles(files)
	if len(s.hashes) != len(uniqueFiles) {
		s.hashes = make(map[string][]byte)
		notChanged = false
	}
	cpus := runtime.NumCPU()
	newHashes := make(map[string][]byte)
	hashMutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(cpus)
	for i := 0; i < cpus; i++ {
		go func(start int) {
			for j := start; j < len(uniqueFiles); j += cpus {
				file := uniqueFiles[j]
				hash := fileHash(file)
				hashMutex.Lock()
				newHashes[file] = hash
				hashMutex.Unlock()
				if existingHash, ok := s.hashes[file]; !ok || !bytes.Equal(hash, existingHash) {
					notChanged = false
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	s.hashes = newHashes
	if !notChanged {
		return !notChanged, &WatchResult{Old: oldHashes, New: newHashes}
	}

	return !notChanged, nil
}

func fileHash(path string) []byte {
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		hash := sha256.New()
		if _, err = io.Copy(hash, f); err == nil {
			return hash.Sum(nil)
		}
	}
	return []byte{}
}

func Watch(timeout time.Duration, files []string) *Session {
	changesChan := make(chan *WatchResult)
	s := &Session{hashes: make(map[string][]byte), lock: &sync.Mutex{}, exit: make(chan interface{}), C: changesChan, files: make(chan []string)}
	timer := time.NewTimer(timeout)
	go func() {
		for {
			select {
			case files = <-s.files:
			case <-s.exit:
				return
			case <-timer.C:
				if changed, watchResult := s.checkFiles(files); changed {
					changesChan <- watchResult
				}
				timer.Reset(timeout)
			}
		}
	}()
	return s
}

func (s *Session) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.exit <- true
}

func (s *Session) UpdateSet(files []string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return
	}
	s.files <- files
}
