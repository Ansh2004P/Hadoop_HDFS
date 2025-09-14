package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
)

func CASPathTransformFunc(key string) pathKey {
	hash := sha1.Sum([]byte(key))          // [20]byte =>slice
	hashStr := hex.EncodeToString(hash[:]) // this trick

	blocksize := 5
	sliceLen := len(hashStr) / blocksize

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashStr[from:to]
	}

	return pathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: key,
	}
}

type pathKey struct {
	Pathname string
	Filename string
}

func (p pathKey) FirstPathName() string {
	paths := strings.Split(p.Pathname, "/") 
	if len(paths)==0 {
		return ""
	}
	return paths[0]
}

func (p pathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.Pathname, p.Filename)
}

type PathTransformFunc func(string) pathKey

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) string {
	return key
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	_, err := os.Stat(pathKey.FullPath())
	if err == fs.ErrNotExist {
		return false
	}
	return true
}


func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from the disk", pathKey.Filename)
	}()

	return os.RemoveAll(pathKey.FirstPathName())
}
func (s *Store) Read(key string) (io.Reader, error) {
	fmt.Printf("Reading file with key: %s\n", key) // Log the key being read

	f, err := s.readStream(key)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return nil, err
	}

	content := buf.String()
	fmt.Printf("File content:\n%s\n", content) // Log the content

	return buf, nil
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	return os.Open(pathKey.FullPath())
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformFunc(key)

	if err := os.MkdirAll(pathKey.Pathname, os.ModePerm); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	io.Copy(buf, r)

	fullPath := pathKey.FullPath()

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, buf)
	if err != nil {
		return err
	}

	log.Printf("written (%d) bytes to disk: %s", n, fullPath)

	return nil
}
