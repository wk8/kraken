package base

import (
	"io"
	"os"
)

// FileReader provides read operation on a file.
type FileReader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

// FileReadWriter provides read/write operation on a file.
type FileReadWriter interface {
	FileReader
	io.Writer
	io.WriterAt

	Size() int64   // required by docker registry.
	Cancel() error // required by docker registry.
	Commit() error // required by docker registry.
}

// FileEntryInternal provides helper functions on one file and its metadata.
// It's not aware of state, and its implementation doesn't guarantee thread safety.
type FileEntryInternal interface {
	GetName() string
	GetPath() string
	Stat() (os.FileInfo, error)

	Create(len int64) error
	CreateLinkFrom(sourcePath string) error
	LinkTo(targetPath string) error
	Move(targetDir string) error
	Delete() error

	AddMetadata(mt MetadataType) error
	ReadMetadata(mt MetadataType) ([]byte, error)
	WriteMetadata(mt MetadataType, data []byte) (bool, error)
	ReadMetadataAt(mt MetadataType, b []byte, off int64) (int, error)
	WriteMetadataAt(mt MetadataType, b []byte, off int64) (int, error)
	DeleteMetadata(mt MetadataType) error
	RangeMetadata(f func(mt MetadataType) error) error
}

// FileEntryInternalFactory initializes FileEntryInternal obj.
type FileEntryInternalFactory interface {
	// Create creates a file entry given a state directory and a name.
	// It calls GetRelativePath to generate the actual file path under given directory,
	Create(dir, name string) FileEntryInternal
	// GetRelativePath returns the relative path for a file entry.
	// The path is relative to the state directory that file entry belongs to.
	// i.e. a file entry can have a relative path of 00/0e/filename under directory /var/cache/
	GetRelativePath(name string) string
}

// FileState decides what directory a file is in.
// A file can only be in one state at any given time.
type FileState interface {
	GetDirectory() string
}

// Verify is passed into FileEntry functions to perform verification after lock is acquired.
type Verify func(entry FileEntry) error

var noopVerify = func(FileEntry) error { return nil }

// FileEntry manages one file and its metadata in a stateful and thread-safe manner.
type FileEntry interface {
	GetInternal() FileEntryInternal
	GetStateUnsafe() FileState
	GetState(v Verify) (FileState, error)
	SetState(v Verify, state FileState) error

	GetName(v Verify) (string, error)
	GetPath(v Verify) (string, error)
	Stat(v Verify) (os.FileInfo, error)

	Create(v Verify, targetState FileState, len int64) error
	CreateLinkFrom(v Verify, targetState FileState, sourcePath string) error
	LinkTo(v Verify, targetPath string) error
	Move(v Verify, targetState FileState) error
	Delete(v Verify) error

	GetReader(v Verify) (FileReader, error)
	GetReadWriter(v Verify) (FileReadWriter, error)

	AddMetadata(v Verify, mt MetadataType) error
	ReadMetadata(v Verify, mt MetadataType) ([]byte, error)
	WriteMetadata(v Verify, mt MetadataType, data []byte) (bool, error)
	ReadMetadataAt(v Verify, mt MetadataType, b []byte, off int64) (int, error)
	WriteMetadataAt(v Verify, mt MetadataType, b []byte, off int64) (int, error)
	DeleteMetadata(v Verify, mt MetadataType) error
	RangeMetadata(v Verify, f func(mt MetadataType) error) error
}

// FileEntryFactory initializes FileEntry obj.
type FileEntryFactory interface {
	Create(state FileState, fi FileEntryInternal) FileEntry
}

// FileMap is a thread-safe name -> FileEntry map.
type FileMap interface {
	Load(key interface{}) (interface{}, bool)
	LoadOrStore(key, value interface{}) (interface{}, bool)
	Delete(key interface{})
}

// FileMapFactory is an interface initilizes a FileMap object.
type FileMapFactory interface {
	Create() (FileMap, error)
}

// FileStore manages files and their metadata in a stateful and thread-safe manner.
type FileStore interface {
	CreateFile(fileName string, states []FileState, createState FileState, len int64) error
	CreateLinkFromFile(fileName string, states []FileState, createState FileState, sourcePath string) error
	LinkToFile(fileName string, states []FileState, targetPath string) error
	MoveFile(fileName string, states []FileState, goalState FileState) error
	DeleteFile(fileName string, states []FileState) error

	GetFilePath(fileName string, states []FileState) (string, error)
	GetFileStat(fileName string, states []FileState) (os.FileInfo, error)

	GetFileReader(fileName string, states []FileState) (FileReader, error)
	GetFileReadWriter(fileName string, states []FileState) (FileReadWriter, error)

	ReadFileMetadata(fileName string, states []FileState, mt MetadataType) ([]byte, error)
	WriteFileMetadata(fileName string, states []FileState, mt MetadataType, data []byte) (bool, error)
	ReadFileMetadataAt(fileName string, states []FileState, mt MetadataType, b []byte, off int64) (int, error)
	WriteFileMetadataAt(fileName string, states []FileState, mt MetadataType, b []byte, off int64) (int, error)
	DeleteFileMetadata(fileName string, states []FileState, mt MetadataType) error
	RangeFileMetadata(fileName string, states []FileState, f func(mt MetadataType) error) error
}

// FileStoreBuilder is used to init new FileStore object.
type FileStoreBuilder interface {
	SetFileEntryInternalFactory(FileEntryInternalFactory) FileStoreBuilder
	SetFileEntryFactory(FileEntryFactory) FileStoreBuilder
	SetFileMapFactory(FileMapFactory) FileStoreBuilder
	Build() (FileStore, error)
}