package scanner

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectserver"
	"io"
	"log"
	"sync"
)

// TODO: the types should probably be moved into a separate package, leaving
//       behind the scanner code.

const metadataFile = ".metadata"

type Object struct {
	length uint64
}

type notifiers map[<-chan string]chan<- string
type makeDirectoryNotifiers map[<-chan image.Directory]chan<- image.Directory

type ImageDataBase struct {
	sync.RWMutex
	// Protected by lock.
	baseDir         string
	directoryMap    map[string]image.DirectoryMetadata
	imageMap        map[string]*image.Image
	addNotifiers    notifiers
	deleteNotifiers notifiers
	mkdirNotifiers  makeDirectoryNotifiers
	// Unprotected by lock.
	objectServer               objectserver.FullObjectServer
	cleanupUnreferencedObjects bool
	logger                     *log.Logger
}

func LoadImageDataBase(baseDir string, objSrv objectserver.FullObjectServer,
	cleanupUnreferencedObjects bool,
	logger *log.Logger) (*ImageDataBase, error) {
	return loadImageDataBase(baseDir, objSrv, cleanupUnreferencedObjects,
		logger)
}

func (imdb *ImageDataBase) AddImage(image *image.Image, name string,
	username *string) error {
	return imdb.addImage(image, name, username)
}

func (imdb *ImageDataBase) CheckImage(name string) bool {
	return imdb.checkImage(name)
}

func (imdb *ImageDataBase) ChownDirectory(dirname, ownerGroup string) error {
	return imdb.chownDirectory(dirname, ownerGroup)
}

func (imdb *ImageDataBase) CountDirectories() uint {
	return imdb.countDirectories()
}

func (imdb *ImageDataBase) CountImages() uint {
	return imdb.countImages()
}

func (imdb *ImageDataBase) DeleteImage(name string, username *string) error {
	return imdb.deleteImage(name, username)
}

// DeleteUnreferencedObjects will delete some or all unreferenced objects.
// Objects are randomly selected for deletion, until both the percentage and
// bytes thresholds are satisfied.
// If an image upload/replication is in process this operation is unsafe as it
// may delete objects that the new image will be using.
func (imdb *ImageDataBase) DeleteUnreferencedObjects(percentage uint8,
	bytes uint64) error {
	return imdb.deleteUnreferencedObjects(percentage, bytes)
}

func (imdb *ImageDataBase) GetImage(name string) *image.Image {
	return imdb.getImage(name)
}

func (imdb *ImageDataBase) ListDirectories() []image.Directory {
	return imdb.listDirectories()
}

func (imdb *ImageDataBase) ListImages() []string {
	return imdb.listImages()
}

// ListUnreferencedObjects will return a map listing all the objects and their
// corresponding sizes which are not referenced by an image.
// Note that some objects may have been recently added and the referencing image
// may not yet be present (i.e. it may be added after missing objects are
// uploaded).
func (imdb *ImageDataBase) ListUnreferencedObjects() map[hash.Hash]uint64 {
	return imdb.listUnreferencedObjects()
}

func (imdb *ImageDataBase) MakeDirectory(dirname, username string) error {
	return imdb.makeDirectory(image.Directory{Name: dirname}, username, true)
}

func (imdb *ImageDataBase) ObjectServer() objectserver.ObjectServer {
	return imdb.objectServer
}

func (imdb *ImageDataBase) RegisterAddNotifier() <-chan string {
	return imdb.registerAddNotifier()
}

func (imdb *ImageDataBase) RegisterDeleteNotifier() <-chan string {
	return imdb.registerDeleteNotifier()
}

func (imdb *ImageDataBase) RegisterMakeDirectoryNotifier() <-chan image.Directory {
	return imdb.registerMakeDirectoryNotifier()
}

func (imdb *ImageDataBase) UnregisterAddNotifier(channel <-chan string) {
	imdb.unregisterAddNotifier(channel)
}

func (imdb *ImageDataBase) UnregisterDeleteNotifier(channel <-chan string) {
	imdb.unregisterDeleteNotifier(channel)
}

func (imdb *ImageDataBase) UnregisterMakeDirectoryNotifier(
	channel <-chan image.Directory) {
	imdb.unregisterMakeDirectoryNotifier(channel)
}

func (imdb *ImageDataBase) UpdateDirectory(directory image.Directory) error {
	return imdb.makeDirectory(directory, "", false)
}

func (imdb *ImageDataBase) WriteHtml(writer io.Writer) {
	imdb.writeHtml(writer)
}
